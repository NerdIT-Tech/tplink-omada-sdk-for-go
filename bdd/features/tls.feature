Feature: TLS handling for self-signed controllers
  As an integrator running Omada on a local network
  I want to choose whether the SDK verifies the controller's TLS certificate
  So that I can connect to self-hosted controllers with self-signed certificates while
  staying secure by default everywhere else

  Scenario: A default HTTP client rejects a self-signed controller certificate
    Given a client configured with the default, certificate-verifying HTTP client
    When the client calls the controller over HTTPS
    Then the request fails with a certificate verification error

  Scenario: An explicitly insecure HTTP client accepts a self-signed controller certificate
    Given a client configured with the SDK's insecure HTTP client
    When the client calls the controller over HTTPS
    Then the request reaches the controller without a certificate verification error
