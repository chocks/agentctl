# agentctl_sdk.TraceApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_traces**](TraceApi.md#list_traces) | **GET** /v1/traces | List or search traces


# **list_traces**
> TraceListResponse list_traces(session_id=session_id, action=action, verdict=verdict, package=package, since=since, until=until, limit=limit)

List or search traces

### Example


```python
import agentctl_sdk
from agentctl_sdk.models.action import Action
from agentctl_sdk.models.trace_list_response import TraceListResponse
from agentctl_sdk.models.verdict import Verdict
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
    api_instance = agentctl_sdk.TraceApi(api_client)
    session_id = 'session_id_example' # str |  (optional)
    action = agentctl_sdk.Action() # Action |  (optional)
    verdict = agentctl_sdk.Verdict() # Verdict |  (optional)
    package = 'package_example' # str |  (optional)
    since = '2013-10-20T19:20:30+01:00' # datetime |  (optional)
    until = '2013-10-20T19:20:30+01:00' # datetime |  (optional)
    limit = 56 # int |  (optional)

    try:
        # List or search traces
        api_response = api_instance.list_traces(session_id=session_id, action=action, verdict=verdict, package=package, since=since, until=until, limit=limit)
        print("The response of TraceApi->list_traces:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TraceApi->list_traces: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **session_id** | **str**|  | [optional] 
 **action** | [**Action**](.md)|  | [optional] 
 **verdict** | [**Verdict**](.md)|  | [optional] 
 **package** | **str**|  | [optional] 
 **since** | **datetime**|  | [optional] 
 **until** | **datetime**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**TraceListResponse**](TraceListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Matching traces |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

