# ReplayRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session_id** | **str** |  | 
**policy_path** | **str** |  | [optional] 
**limit** | **int** |  | [optional] 

## Example

```python
from agentctl_sdk.models.replay_request import ReplayRequest

# TODO update the JSON string below
json = "{}"
# create an instance of ReplayRequest from a JSON string
replay_request_instance = ReplayRequest.from_json(json)
# print the JSON string representation of the object
print(ReplayRequest.to_json())

# convert the object into a dict
replay_request_dict = replay_request_instance.to_dict()
# create an instance of ReplayRequest from a dict
replay_request_from_dict = ReplayRequest.from_dict(replay_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


