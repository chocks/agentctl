# RunCodeParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**language** | **str** |  | 
**command** | **str** |  | 
**stdin** | **str** |  | [optional] 
**network** | **bool** |  | [optional] 

## Example

```python
from agentctl_sdk.models.run_code_params import RunCodeParams

# TODO update the JSON string below
json = "{}"
# create an instance of RunCodeParams from a JSON string
run_code_params_instance = RunCodeParams.from_json(json)
# print the JSON string representation of the object
print(RunCodeParams.to_json())

# convert the object into a dict
run_code_params_dict = run_code_params_instance.to_dict()
# create an instance of RunCodeParams from a dict
run_code_params_from_dict = RunCodeParams.from_dict(run_code_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


