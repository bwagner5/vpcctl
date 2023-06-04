# VPC-CTL

VPC CTL is a simple CLI tool to create AWS Virtual Private Clouds (VPC). 

## Why? 

VPCs are the foundation of most cloud deployments. Development in the cloud often requires creating VPCs for testing, especially when using Kubernetes where clusters are often 1:1 with a VPC. 

VPCs are also the most straight forward thing to setup. There are many sub-resources of a VPC that are required to make everything work. 

VPC CTL aims to make creating VPCs a breeze.

## Usage:

```
Usage:
  vpcctl [command]

Available Commands:
  create      Create a VPC
  delete      Delete a VPC
  get         Get a VPC
  list        List VPCs
  help        Help about any command

Flags:
  -f, --file string   YAML Config File
  -h, --help          help for vpcctl
      --verbose       Verbose output
      --version       version

Use "vpcctl [command] --help" for more information about a command.
```
## Installation:

```
brew install bwagner5/wagner/vpcctl
```

Packages, binaries, and archives are published for all major platforms (Mac amd64/arm64 & Linux amd64/arm64):

Debian / Ubuntu:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget https://github.com/bwagner5/vpcctl/releases/download/v0.0.4/vpcctl_0.0.4_${OS}_${ARCH}.deb
dpkg --install vpcctl_0.0.2_linux_amd64.deb
vpcctl --help
```

RedHat:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
rpm -i https://github.com/bwagner5/vpcctl/releases/download/v0.0.4/vpcctl_0.0.4_${OS}_${ARCH}.rpm
```

Download Binary Directly:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget -qO- https://github.com/bwagner5/vpcctl/releases/download/v0.0.4/vpcctl_0.0.4_${OS}_${ARCH}.tar.gz | tar xvz
chmod +x vpcctl
```

## Examples: 

```
> vpcctl create --name my-test-vpc
2023/06/03 15:05:24 Creating VPC test-vpc
2023/06/03 15:05:25 Created VPC vpc-0503ee50c5024dcb7
2023/06/03 15:05:25 Creating Subnets
2023/06/03 15:05:27 Created Public Subnets [subnet-0c1bdb211bd9c8523 subnet-0f72ebf8cd00ce550 subnet-08e515bdadefd4a82]
2023/06/03 15:05:27 Created Private Subnets [subnet-0bc4fb64c508e78de subnet-0bf5550c42a28a5fc subnet-09c56c49f329c985f]
2023/06/03 15:05:27 Creating Route Tables
2023/06/03 15:05:28 Created Route Tables: [rtb-0a008554f7841d6d3 rtb-0592762d7fefb1194]
2023/06/03 15:05:28 Creating Internet Gateway
2023/06/03 15:05:29 Created Internet Gateway: igw-0b41651158051ad34
2023/06/03 15:05:29 Creating NAT Gateway
2023/06/03 15:08:28 Created NAT Gateway: nat-03e798e0ef0f10359
{
    "VPC": {
        "CidrBlock": "10.0.0.0/16",
        "CidrBlockAssociationSet": [
            {
                "AssociationId": "vpc-cidr-assoc-06ff9713eeecc84e2",
                "CidrBlock": "10.0.0.0/16",
                "CidrBlockState": {
                    "State": "associated",
                    "StatusMessage": null
                }
            }
        ],
...
```

```
> vpcctl get --name my-test-vpc
{
    "VPC": {
        "CidrBlock": "10.0.0.0/16",
        "CidrBlockAssociationSet": [
            {
                "AssociationId": "vpc-cidr-assoc-06ff9713eeecc84e2",
                "CidrBlock": "10.0.0.0/16",
                "CidrBlockState": {
                    "State": "associated",
                    "StatusMessage": null
                }
            }
        ],
...

```

```
> vpcctl list
my-test-vpc

```

```
> vpcctl delete
2023/06/03 15:08:41 Fetching VPC details for test-vpc
2023/06/03 15:08:42 Deleting NAT Gateway nat-03e798e0ef0f10359
2023/06/03 15:09:56 Deleted NAT Gateway nat-03e798e0ef0f10359
2023/06/03 15:09:56 Deleting Internet Gateway igw-0b41651158051ad34
2023/06/03 15:09:56 Deleted Internet Gateway igw-0b41651158051ad34
2023/06/03 15:09:56 Deleting Route Tables [rtb-0a008554f7841d6d3 rtb-0592762d7fefb1194]
2023/06/03 15:09:58 Deleted Route Tables [rtb-0a008554f7841d6d3 rtb-0592762d7fefb1194]
2023/06/03 15:09:58 Deleting Subnets [subnet-0f72ebf8cd00ce550 subnet-0c1bdb211bd9c8523 subnet-09c56c49f329c985f subnet-0bf5550c42a28a5fc subnet-0bc4fb64c508e78de subnet-08e515bdadefd4a82]
2023/06/03 15:10:00 Deleted Subnets [subnet-0f72ebf8cd00ce550 subnet-0c1bdb211bd9c8523 subnet-09c56c49f329c985f subnet-0bf5550c42a28a5fc subnet-0bc4fb64c508e78de subnet-08e515bdadefd4a82]
2023/06/03 15:10:00 Deleting VPC vpc-0503ee50c5024dcb7
Deleted VPC test-vpc
```