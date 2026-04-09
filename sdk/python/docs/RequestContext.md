# RequestContext


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session_id** | **str** |  | 
**model** | **str** |  | [optional] 
**agent** | **str** |  | [optional] 
**turn** | **int** |  | [optional] 
**timestamp** | **datetime** |  | 

## Example

```python
from agentctl_sdk.models.request_context import RequestContext

# TODO update the JSON string below
json = "{}"
# create an instance of RequestContext from a JSON string
request_context_instance = RequestContext.from_json(json)
# print the JSON string representation of the object
print(RequestContext.to_json())

# convert the object into a dict
request_context_dict = request_context_instance.to_dict()
# create an instance of RequestContext from a dict
request_context_from_dict = RequestContext.from_dict(request_context_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


