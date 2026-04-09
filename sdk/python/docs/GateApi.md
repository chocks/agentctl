# agentctl_sdk.GateApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**gate_action**](GateApi.md#gate_action) | **POST** /v1/gate | Evaluate a risky action against policy


# **gate_action**
> Decision gate_action(gate_request)

Evaluate a risky action against policy

### Example


```python
import agentctl_sdk
from agentctl_sdk.models.decision import Decision
from agentctl_sdk.models.gate_request import GateRequest
from agentctl_sdk.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8080
# See configuration.py for a list of all supported configuration parameters.
configuration = agentctl_sdk.Configuration(
    host = "http://localhost:8080"
)


# Enter a context with an instance of the API client
with agentctl_sdk.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = agentctl_sdk.GateApi(api_client)
    gate_request = agentctl_sdk.GateRequest() # GateRequest | 

    try:
        # Evaluate a risky action against policy
        api_response = api_instance.gate_action(gate_request)
        print("The response of GateApi->gate_action:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling GateApi->gate_action: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gate_request** | [**GateRequest**](GateRequest.md)|  | 

### Return type

[**Decision**](Decision.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Decision returned |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

