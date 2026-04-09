# agentctl_sdk.ReplayApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**replay_session**](ReplayApi.md#replay_session) | **POST** /v1/replay | Re-evaluate a recorded session with a given policy


# **replay_session**
> ReplayResponse replay_session(replay_request)

Re-evaluate a recorded session with a given policy

### Example


```python
import agentctl_sdk
from agentctl_sdk.models.replay_request import ReplayRequest
from agentctl_sdk.models.replay_response import ReplayResponse
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
    api_instance = agentctl_sdk.ReplayApi(api_client)
    replay_request = agentctl_sdk.ReplayRequest() # ReplayRequest | 

    try:
        # Re-evaluate a recorded session with a given policy
        api_response = api_instance.replay_session(replay_request)
        print("The response of ReplayApi->replay_session:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ReplayApi->replay_session: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **replay_request** | [**ReplayRequest**](ReplayRequest.md)|  | 

### Return type

[**ReplayResponse**](ReplayResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Replay results |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

