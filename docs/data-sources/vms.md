# xenorchestra_vms

Use this data source to filter Xenorchestra VMs by certain criteria (pool_id, power_state or host) for use in other resources.

## Example Usage

```hcl
data "xenorchestra_pool" "pool" {
  name_label = "Your pool"
}

data "xenorchestra_vms" "vms" {
  pool_id = data.xenorchestra_pool.pool.id
  power_state = "Running"
  host = data.xenorchestra_pool.pool.master
}

output "vms_max_memory_map" {
  value = tomap({
  for k, vm in data.xenorchestra_vms.vms.vms : k => vm.memory_max
  })
}
output "vms_length" {
  value = length(data.xenorchestra_vms.vms.vms)
}
```

## Argument Reference

* pool_id - (Required) The ID of the pool the vms belong to.
* host - (Optional) The Id of the host (container) the vms belong to.
* power_state - (Optional) The power state of the vms ("Running" / "Halted")

## Attributes Reference

* id - The Id of the pool the storage repository exists on.
* pool_id - The Id of the pool the storage repository exists on.
* vms - A list of information for all vms found in this pool.
    * vms.id - The uuid for this vm.
    * vms.name_label - The name label for this vm.
    * vms.cpu - The number of cpu assigned to this vm.
    * vms.cloud_config - The cloud configuration for this vm.
    * vms.cloud_network_config - The cloud network configuration for this vm.
    * vms.tags - The tags assigned to this vm.
    * vms.memory_max - The maximum memory size for this vm.
    * vms.affinity_host - The affinity host for this vm.
    * vms.template - The template used to create this vm.
    * vms.wait_for_ip - The wait for ip options for this vm.
    * vms.high_availability - The high availability option for this vm.
    * vms.resource_set - The resource sets for this vm.
    * vms.disk - The disk information for this vm.
    * vms.network - The network information for this vm.
    
    
    
    
    
