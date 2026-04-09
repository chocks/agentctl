# WriteFileParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **str** |  | 
**operation** | **str** |  | 
**size_bytes** | **int** |  | [optional] 

## Example

```python
from agentctl_sdk.models.write_file_params import WriteFileParams

# TODO update the JSON string below
json = "{}"
# create an instance of WriteFileParams from a JSON string
write_file_params_instance = WriteFileParams.from_json(json)
# print the JSON string representation of the object
print(WriteFileParams.to_json())

# convert the object into a dict
write_file_params_dict = write_file_params_instance.to_dict()
# create an instance of WriteFileParams from a dict
write_file_params_from_dict = WriteFileParams.from_dict(write_file_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


