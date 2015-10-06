module "lattice-openstack" {
    source = "github.com/hpcloud/lattice//terraform//hpcloud?ref=hpcloud-v0.3.3"

    # OpenStack User Account (your hpcloud.com username)
    openstack_access_key = "<CHANGE-ME>"

    # OpenStack Password
    openstack_secret_key = "<CHANGE-ME>"

    # OpenStack Tenant Name
    openstack_tenant_name = "<CHANGE-ME>"

    # Security Group Name (do not use an existing security group, a new one
    # will be created for you)
    openstack_secgroup = "lattice-sg"

    # SSH Key Name (do not use an existing key-name, a new one will be created
    # for you)
    openstack_key_name = "<CHANGE-ME>"

    # SSH Public Key to Upload
    openstack_public_key = "<CHANGE-ME>"

    # Path & filename of the SSH private key file
    openstack_ssh_private_key_file = "<CHANGE-ME>"

    # The number of Lattice Cells to launch
    num_cells = "1"

    # Lattice Username
    lattice_username = "user"

    # Lattice Password
    lattice_password = "pass"

    #################################
    ###  Optional Settings Below  ###
    #################################

    # This is the user used to login to VM instances via SSH 
    openstack_ssh_user = "ubuntu"

    # URI of Keystone authentication agent
    # You shouldn't need to change this setting when deploying to hpcloud.com
    openstack_keystone_uri = "https://region-a.geo-1.identity.hpcloudsvc.com:35357/v2.0/"

    # Instance Flavor Types
    openstack_instance_type_coordinator = "standard.medium"
    openstack_instance_type_cell = "standard.medium"

    # The internet-facing network which Neutron L3 routers should use as a gateway (UUID)
    # You shouldn't need to change this setting when deploying to hpcloud.com
    openstack_neutron_router_gateway_network_id = "7da74520-9d5e-427b-a508-213c84e69616"

    # The name of the pool that floating IP addresses will be requested from
    # You shouldn't need to change this setting when deploying to hpcloud.com
    openstack_floating_ip_pool_name = "Ext-Net"

    # The name of the Openstack Glance image used to spin up all VM instances.
    openstack_image = "Ubuntu Server 14.04.1 LTS (amd64 20150706) - Partner Image"

    # If you wish to use your own lattice release instead of the latest version,
    # uncomment the variable assignment below and set it to your own lattice tar's
    # path.
    # local_lattice_tar_path = "~/lattice.tgz"

    # Openstack Region (Blank default for 'no region' installations)
    # The default is US East, use region-a.geo-1 for US West
    openstack_region = "region-b.geo-1"
}

output "lattice_target" {
    value = "${module.lattice-openstack.lattice_target}"
}

output "lattice_username" {
    value = "${module.lattice-openstack.lattice_username}"
}

output "lattice_password" {
    value = "${module.lattice-openstack.lattice_password}"
}
