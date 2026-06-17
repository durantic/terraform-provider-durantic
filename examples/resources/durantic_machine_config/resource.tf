variable "gateway_hostname" {
  type    = string
  default = "gateway-01"
}

data "durantic_machine" "gateway" {
  hostname = var.gateway_hostname
}

data "durantic_image" "rke2_server" {
  docker_image_url = "ghcr.io/durantic/linux-ubuntu-25.10:rke2-server-1.35"
}

resource "durantic_mesh_network" "cluster" {
  name         = "rke2-cluster-mesh"
  network_cidr = "10.50.0.0/24"
}

resource "durantic_variable" "gateway_public_ip" {
  name  = "DEMO_GATEWAY_PUBLIC_IP"
  value = data.durantic_machine.gateway.public_ip_addresses[0]
}

resource "durantic_machine_role" "gateway" {
  name          = "demo-gateway-haproxy-k8s"
  description   = "HAProxy gateway role for an RKE2 cluster"
  requires_mesh = true

  template_data = <<-EOT
    #cloud-config
    packages:
      - haproxy
  EOT
}

resource "durantic_machine_role" "rke2_server" {
  name          = "demo-rke2-server"
  description   = "RKE2 server role"
  image_uuid    = data.durantic_image.rke2_server.uuid
  requires_mesh = true

  template_data = <<-EOT
    #cloud-config
    runcmd:
      - echo "RKE2 server role applied"
  EOT
}

resource "durantic_machine_config" "gateway" {
  machine_uuid      = data.durantic_machine.gateway.uuid
  mesh_network_uuid = durantic_mesh_network.cluster.uuid

  role_names = [
    durantic_machine_role.gateway.name,
  ]
}
