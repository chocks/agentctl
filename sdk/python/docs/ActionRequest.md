# ActionRequest

Stored request shape inside a recorded decision.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | [**Action**](Action.md) |  | 
**params** | [**ActionParams**](ActionParams.md) |  | 
**reason** | **str** |  | 
**context** | [**RequestContext**](RequestContext.md) |  | [optional] 

## Example

```python
from agentctl_sdk.models.action_request import ActionRequest

# TODO update the JSON string below
json = "{}"
# create an instance of ActionRequest from a JSON string
action_request_instance = ActionRequest.from_json(json)
# print the JSON string representation of the object
print(ActionRequest.to_json())

# convert the object into a dict
action_request_dict = action_request_instance.to_dict()
# create an instance of ActionRequest from a dict
action_request_from_dict = ActionRequest.from_dict(action_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


