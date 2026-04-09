# CallExternalAPIParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**url** | **str** |  | 
**method** | **str** |  | 
**domain** | **str** |  | [optional] 

## Example

```python
from agentctl_sdk.models.call_external_api_params import CallExternalAPIParams

# TODO update the JSON string below
json = "{}"
# create an instance of CallExternalAPIParams from a JSON string
call_external_api_params_instance = CallExternalAPIParams.from_json(json)
# print the JSON string representation of the object
print(CallExternalAPIParams.to_json())

# convert the object into a dict
call_external_api_params_dict = call_external_api_params_instance.to_dict()
# create an instance of CallExternalAPIParams from a dict
call_external_api_params_from_dict = CallExternalAPIParams.from_dict(call_external_api_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


