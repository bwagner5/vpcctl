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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func (v Client) deleteNATGW(ctx context.Context, vpcDetails *Details, _ DeleteOptions) error {
	if _, err := v.ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{NatGatewayId: vpcDetails.NATGateway.NatGatewayId}); err != nil {
		return err
	}
	waiter := ec2.NewNatGatewayDeletedWaiter(v.ec2Client)
	if err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{NatGatewayIds: []string{*vpcDetails.NATGateway.NatGatewayId}}, 5*time.Minute); err != nil {
		return err
	}
	for _, eipAllocation := range vpcDetails.NATGateway.NatGatewayAddresses {
		if _, err := v.ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: eipAllocation.AllocationId}); err != nil {
			return err
		}
	}
	return nil
}

func (v Client) deleteIGW(ctx context.Context, vpcDetails *Details, _ DeleteOptions) error {
	if _, err := v.ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{InternetGatewayId: vpcDetails.InternetGateway.InternetGatewayId, VpcId: vpcDetails.VPC.VpcId}); err != nil {
		return err
	}
	if _, err := v.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{InternetGatewayId: vpcDetails.InternetGateway.InternetGatewayId}); err != nil {
		return err
	}
	return nil
}

func (v Client) deleteRouteTables(ctx context.Context, vpcDetails *Details, _ DeleteOptions) error {
	for _, rt := range vpcDetails.RouteTables {
		for _, route := range rt.Routes {
			if route.GatewayId != nil && strings.HasPrefix(*route.GatewayId, "igw-") {
				if _, err := v.ec2Client.DeleteRoute(ctx, &ec2.DeleteRouteInput{RouteTableId: rt.RouteTableId, DestinationCidrBlock: route.DestinationCidrBlock}); err != nil {
					return err
				}
			}
		}
		for _, association := range rt.Associations {
			if _, err := v.ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{AssociationId: association.RouteTableAssociationId}); err != nil {
				return err
			}
		}
		if _, err := v.ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{RouteTableId: rt.RouteTableId}); err != nil {
			return err
		}
	}
	return nil
}

func (v Client) deleteSubnets(ctx context.Context, vpcDetails *Details, _ DeleteOptions) error {
	for _, subnet := range vpcDetails.Subnets {
		if _, err := v.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: subnet.SubnetId}); err != nil {
			return err
		}
	}
	return nil
}

func (v Client) deleteVPC(ctx context.Context, vpcDetails *Details, _ DeleteOptions) error {
	if _, err := v.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: vpcDetails.VPC.VpcId}); err != nil {
		return err
	}
	return nil
}
