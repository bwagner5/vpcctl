/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpc

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/samber/lo"
)

func (v Client) createVPC(ctx context.Context, opts CreateOptions) (*types.Vpc, error) {
	vpcOut, err := v.ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: &opts.CIDR,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: lo.Flatten([][]types.Tag{
					defaultTags,
					{
						{Key: aws.String("Name"), Value: &opts.Name},
					},
					v.userTags(opts),
				}),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return vpcOut.Vpc, nil
}

func (v Client) createSubnets(ctx context.Context, vpcID string, opts CreateOptions) ([]*types.Subnet, error) {
	var subnetOutputs []*ec2.CreateSubnetOutput
	// Create subnets
	for _, subnet := range opts.Subnets {
		subnetType := lo.Ternary(subnet.Public, SubnetTypePublic, SubnetTypePrivate)
		subnetOutput, err := v.ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            &vpcID,
			AvailabilityZone: &subnet.AZ,
			CidrBlock:        &subnet.CIDR,
			TagSpecifications: []types.TagSpecification{{
				ResourceType: types.ResourceTypeSubnet,
				Tags: lo.Flatten([][]types.Tag{
					defaultTags,
					{
						{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%s-%s", opts.Name, subnet.AZ, subnetType))},
						{Key: aws.String("Type"), Value: &subnetType},
					},
					v.userTags(opts),
				}),
			},
			},
		})
		if err != nil {
			return nil, err
		}
		if subnetType == SubnetTypePublic {
			subnetOutput.Subnet.MapPublicIpOnLaunch = aws.Bool(true)
		}
		subnetOutputs = append(subnetOutputs, subnetOutput)
	}
	// Modify any subnet attributes that we can't set on creation
	for _, subnet := range subnetOutputs {
		subnetOpts, ok := lo.Find(opts.Subnets, func(subnetOpts CreateSubnetOptions) bool { return subnetOpts.CIDR == *subnet.Subnet.CidrBlock })
		if !ok {
			return nil, fmt.Errorf("unable to find SubnetCreationOptions for subnet %s - %s", *subnet.Subnet.AvailabilityZone, *subnet.Subnet.CidrBlock)
		}
		// Can only modify 1 subnet attribute at a time
		if subnetOpts.Public {
			if _, err := v.ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
				SubnetId:            subnet.Subnet.SubnetId,
				MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
			}); err != nil {
				return nil, err
			}
		}
	}
	return lo.Map(subnetOutputs, func(out *ec2.CreateSubnetOutput, _ int) *types.Subnet { return out.Subnet }), nil
}

func (v Client) createRouteTables(ctx context.Context, subnets []*types.Subnet, opts CreateOptions) (map[string]*types.RouteTable, error) {
	privateSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return !*subnet.MapPublicIpOnLaunch })
	publicSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return *subnet.MapPublicIpOnLaunch })
	routeTables := map[string]*types.RouteTable{}
	// PUBLIC SUBNET RESOURCES
	var publicRouteTableOut *ec2.CreateRouteTableOutput
	for i, publicSubnet := range publicSubnets {
		if i == 0 {
			var err error
			publicRouteTableOut, err = v.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
				VpcId: publicSubnet.VpcId,
				TagSpecifications: []types.TagSpecification{
					{
						ResourceType: types.ResourceTypeRouteTable,
						Tags: lo.Flatten([][]types.Tag{
							defaultTags,
							{
								{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%s", opts.Name, "PUBLIC"))},
							},
							v.userTags(opts),
						}),
					},
				},
			})
			if err != nil {
				return nil, err
			}
			routeTables[SubnetTypePublic] = publicRouteTableOut.RouteTable
		}
		if _, err := v.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
			RouteTableId: publicRouteTableOut.RouteTable.RouteTableId,
			SubnetId:     publicSubnet.SubnetId,
		}); err != nil {
			return nil, err
		}
	}

	// PRIVATE SUBNET RESOURCES
	var privateRouteTableOut *ec2.CreateRouteTableOutput
	for i, privateSubnet := range privateSubnets {
		if i == 0 {
			var err error
			privateRouteTableOut, err = v.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
				VpcId: privateSubnet.VpcId,
				TagSpecifications: []types.TagSpecification{
					{
						ResourceType: types.ResourceTypeRouteTable,
						Tags: lo.Flatten([][]types.Tag{
							defaultTags,
							{
								{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%s", opts.Name, "PRIVATE"))},
							},
							v.userTags(opts),
						}),
					},
				},
			})
			if err != nil {
				return nil, err
			}
			routeTables[SubnetTypePrivate] = privateRouteTableOut.RouteTable
		}
		if _, err := v.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
			RouteTableId: privateRouteTableOut.RouteTable.RouteTableId,
			SubnetId:     privateSubnet.SubnetId,
		}); err != nil {
			return nil, err
		}
	}
	return routeTables, nil
}

func (v Client) createNATGW(ctx context.Context, subnets []*types.Subnet, routeTable *types.RouteTable, opts CreateOptions) (*types.NatGateway, error) {
	// privateSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return !*subnet.MapPublicIpOnLaunch })
	publicSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return *subnet.MapPublicIpOnLaunch })
	eipOut, err := v.ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeElasticIp,
				Tags: lo.Flatten([][]types.Tag{
					defaultTags,
					{
						{Key: aws.String("Name"), Value: &opts.Name},
					},
					v.userTags(opts),
				}),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	natGWOut, err := v.ec2Client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
		AllocationId: eipOut.AllocationId,
		SubnetId:     publicSubnets[0].SubnetId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeNatgateway,
				Tags: lo.Flatten([][]types.Tag{
					defaultTags,
					{
						{Key: aws.String("Name"), Value: &opts.Name},
					},
					v.userTags(opts),
				}),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	waiter := ec2.NewNatGatewayAvailableWaiter(v.ec2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{NatGatewayIds: []string{*natGWOut.NatGateway.NatGatewayId}}, 5*time.Minute); err != nil {
		return natGWOut.NatGateway, err
	}
	if _, err := v.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         routeTable.RouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		NatGatewayId:         natGWOut.NatGateway.NatGatewayId,
	}); err != nil {
		return nil, err
	}
	return natGWOut.NatGateway, nil
}

func (v Client) createIGW(ctx context.Context, vpcID string, routeTable *types.RouteTable, opts CreateOptions) (*types.InternetGateway, error) {
	igwOut, err := v.ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: lo.Flatten([][]types.Tag{
					defaultTags,
					{
						{Key: aws.String("Name"), Value: &opts.Name},
					},
					v.userTags(opts),
				}),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if _, err := v.ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: igwOut.InternetGateway.InternetGatewayId,
		VpcId:             &vpcID,
	}); err != nil {
		return nil, err
	}
	if _, err := v.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         routeTable.RouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            igwOut.InternetGateway.InternetGatewayId,
	}); err != nil {
		return nil, err
	}
	return igwOut.InternetGateway, nil
}

func (v Client) userTags(opts CreateOptions) []types.Tag {
	return lo.MapToSlice(opts.Tags, func(k string, v string) types.Tag {
		return types.Tag{
			Key:   &k,
			Value: &v,
		}
	})
}
