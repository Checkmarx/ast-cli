namespace EvidenceResolver.Tests.Contract
{
    public static class MockProviderServiceExtenstion
    {
        public static IMockProviderService WithRequest(this IMockProviderService mockProviderService,
            HttpVerb method, object path, object body = null, Dictionary<string, object> headers = null)
        {
            var providerServiceRequest = new ProviderServiceRequest
            {
                Method = method,
                Path = path
            };

            providerServiceRequest.Headers = headers ?? new Dictionary<string, object>
            {
                {"Content-Type", "application/json"}
            };

            if (body != null) {
                providerServiceRequest.Body = PactNet.Matchers.Match.Type(body);
            }

            return mockProviderService.With(providerServiceRequest);
        }

        public static void WillRespondParameters(this IMockProviderService mockProviderService,
            int status, dynamic body = null, Dictionary<string, object> headers = null)
        {
            if (body == null) {
                body = new { };
            }

            var expectedResponse = new ProviderServiceResponse
            {
                Status = status,
                Headers = headers ?? new Dictionary<string, object>
                    {{"Content-Type", "application/json; charset=utf-8"}},
                Body = PactNet.Matchers.Match.Type(body)
            };

            mockProviderService.WillRespondWith(expectedResponse);
        }
    }
}