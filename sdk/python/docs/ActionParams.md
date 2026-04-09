# ActionParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**manager** | **str** |  | 
**package** | **str** |  | 
**version** | **str** |  | [optional] 
**hash** | **str** |  | [optional] 
**pinned** | **bool** |  | [optional] 
**language** | **str** |  | 
**command** | **str** |  | 
**stdin** | **str** |  | [optional] 
**network** | **bool** |  | [optional] 
**name** | **str** |  | 
**scope** | **str** |  | [optional] 
**ttl** | **int** |  | [optional] 
**path** | **str** |  | 
**operation** | **str** |  | 
**size_bytes** | **int** |  | [optional] 
**url** | **str** |  | 
**method** | **str** |  | 
**domain** | **str** |  | [optional] 

## Example

```python
from agentctl_sdk.models.action_params import ActionParams

# TODO update the JSON string below
json = "{}"
# create an instance of ActionParams from a JSON string
action_params_instance = ActionParams.from_json(json)
# print the JSON string representation of the object
print(ActionParams.to_json())

# convert the object into a dict
action_params_dict = action_params_instance.to_dict()
# create an instance of ActionParams from a dict
action_params_from_dict = ActionParams.from_dict(action_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


