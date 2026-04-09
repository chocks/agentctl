# AccessSecretParams


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | 
**scope** | **str** |  | [optional] 
**ttl** | **int** |  | [optional] 

## Example

```python
from agentctl_sdk.models.access_secret_params import AccessSecretParams

# TODO update the JSON string below
json = "{}"
# create an instance of AccessSecretParams from a JSON string
access_secret_params_instance = AccessSecretParams.from_json(json)
# print the JSON string representation of the object
print(AccessSecretParams.to_json())

# convert the object into a dict
access_secret_params_dict = access_secret_params_instance.to_dict()
# create an instance of AccessSecretParams from a dict
access_secret_params_from_dict = AccessSecretParams.from_dict(access_secret_params_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


