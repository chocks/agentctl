# GateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | [**Action**](Action.md) |  | 
**params** | [**ActionParams**](ActionParams.md) |  | 
**reason** | **str** |  | 
**context** | [**RequestContext**](RequestContext.md) |  | [optional] 

## Example

```python
from agentctl_sdk.models.gate_request import GateRequest

# TODO update the JSON string below
json = "{}"
# create an instance of GateRequest from a JSON string
gate_request_instance = GateRequest.from_json(json)
# print the JSON string representation of the object
print(GateRequest.to_json())

# convert the object into a dict
gate_request_dict = gate_request_instance.to_dict()
# create an instance of GateRequest from a dict
gate_request_from_dict = GateRequest.from_dict(gate_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


