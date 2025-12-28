resource "durantic_mesh_network" "example" {
  name         = "example-network"
  network_cidr = "10.0.0.0/16"
  is_default   = false
}
