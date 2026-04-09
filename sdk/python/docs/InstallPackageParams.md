# InstallPackageParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**manager** | **str** |  | 
**package** | **str** |  | 
**version** | **str** |  | [optional] 
**hash** | **str** |  | [optional] 
**pinned** | **bool** |  | [optional] 

## Example

```python
from agentctl_sdk.models.install_package_params import InstallPackageParams

# TODO update the JSON string below
json = "{}"
# create an instance of InstallPackageParams from a JSON string
install_package_params_instance = InstallPackageParams.from_json(json)
# print the JSON string representation of the object
print(InstallPackageParams.to_json())

# convert the object into a dict
install_package_params_dict = install_package_params_instance.to_dict()
# create an instance of InstallPackageParams from a dict
install_package_params_from_dict = InstallPackageParams.from_dict(install_package_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


