# Healthchecks

## Adding a healthcheck

-   Create a function that implements the CheckStatus type (func(name string) \*Status)
    -   The name that is sent in will be the name set on Registration
    -   Return a pointer to the Status object created in teh function
-   Register the healthcheck by calling RegisterHealthcheck providing a name, endpoint, and the CheckStatus function
-   this function will also add an endpoint to the http library at /status/[endpoint]
-   Healthcheck is now registered

## Starting the healthcheck server

-   Within the agent definition make a call to HandleRequests
    -   If a new HTTP server should be started provide a port number greater than 0
    -   If the HTTP server should not be started provide a 0 as the port number
-   This method will register the /status endpoint with the http library

## Check all healthchecks

-   Call the RunChecks function, which will return either an OK or FAIL status

## Wait for all healthchecks to Pass

-   Call the WaitForReady function, once it returns all healthchecks have passed
