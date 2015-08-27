# IMPORTANT

> The terraform template needs to create an OpenStack router and a few Security Group rules.
> By default, you are only allowed one router on hpcloud.com.
> E-mail billing@hpcloud.com to request an increase.

> Sample request e-mail:
> ```
> Hello,
> 
> I need to request a router limit increase to 5 and a security group rule limit increase to 50 in [INSERT REGION].
> 
> Account Information
> 
> *   Name: [INSERT NAME]
> *   Username: [INSERT USERNAME]
> *   Email Address: [INSERT YOUR E-MAIL]	
> *   Project ID: [INSERT YOUR PROJECT/TENANT ID]
> *   Use Case: deploying lattice for development and testing purposes
> *   What Region & Service do you need the increase for: [INSERT REGION]
> 
> Thank you,
> [INSERT NAME]
> ```

  
# Lattice Terraform templates for HP Helion Public Cloud

This project contains [Terraform](https://www.terraform.io/) templates to help you deploy
[Lattice](https://github.com/cloudfoundry-incubator/lattice) on
[HP Helion Public Cloud](http://www.hpcloud.com/). 

## Usage

### Prerequisites

* An [hpcloud.com](http://www.hpcloud.com/) account.
* [Terraform](https://www.terraform.io/downloads.html)

### Configure

Here are some step-by-step instructions for configuring a Lattice cluster via Terraform:

1. Download [`lattice.openstack.tf`](https://github.com/hpcloud/lattice/raw/hpcloud-v0.3.3/terraform/hpcloud/example/lattice.openstack.tf)
2. Create an empty folder and place the `lattice.openstack.tf` file in that folder.
3. Update the `lattice.openstack.tf` by filling in the values for the variables.  Details for the values of those variables are below.

The variables you should configure are:

* `openstack_access_key`: Your hpcloud.com username
* `openstack_secret_key`: Your hpcloud.com password
* `openstack_tenant_name`: Your tenant/project name
* `openstack_key_name`: The Key-Name given to the public key that will be uploaded for use by the VM instances.
* `openstack_public_key`: The actual contents of the public key to upload.
* `openstack_ssh_private_key_file`: Path to the SSH private key file (Stays local. Used for provisioning.)
* `num_cells`: The number of Lattice cells to launch
* `lattice_username`: Lattice username (default `user`)
* `lattice_password`: Lattice password (default `pass`)

> There are other settings available (with descriptions) in the `lattice.openstack.tf` file.
> The defaults for these are specific to hpcloud.com. Please read the descriptions carefully before changing any of them.

### Deploy

Here are some step-by-step instructions for deploying a Lattice cluster via Terraform:

1. Run the following commands in the folder containing the `lattice.openstack.tf` file

  ```bash
  terraform get -update
  terraform apply
  ```

  This will deploy the cluster.

Upon success, terraform will print the Lattice target:

```
Outputs:

  lattice_target = x.x.x.x.xip.io
  lattice_username = xxxxxxxx
  lattice_password = xxxxxxxx
```

which you can use with the Lattice CLI to `ltc target x.x.x.x.xip.io`.

Terraform will generate a `terraform.tfstate` file.  This file describes the cluster that was built - keep it around in order to modify/tear down the cluster.

### Use

Refer to the [Lattice CLI](../../ltc) documentation.

### Destroy

Destroy the cluster:

```
terraform destroy
```

Sometimes, destroy will need to be run twice to completely destroy all components, as openstack networking components are often seen as 'still in use' when destroyed immediately after the instances that relied on them.


## Updating

There is currently no support for updating hocloud.com deployments of lattice with terraform. 

