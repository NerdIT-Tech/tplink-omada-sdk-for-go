Feature: Client credential authentication
  As an integrator embedding the Omada SDK in an application
  I want to authenticate against a real Omada controller using a client ID and secret
  So that my application can call the Open API without managing raw HTTP/OAuth details itself

  Background:
    Given an Omada controller reachable at the configured base URL
    And the controller's Omada Cloud Controller ID

  Scenario: Exchanging valid client credentials for an access token
    Given a client configured with the controller's client ID and client secret
    When the client authenticates with the controller
    Then the controller issues an access token
    And the SDK attaches that access token to the next request using Omada's access token header scheme

  Scenario: Rejecting an authentication attempt with an invalid client secret
    Given a client configured with the controller's client ID and an invalid client secret
    When the client authenticates with the controller
    Then the controller reports an authentication error
    And the SDK surfaces that error to the caller instead of a generic transport failure

  Scenario: Rejecting an authentication attempt with an unknown client ID
    Given a client configured with an unknown client ID and the controller's client secret
    When the client authenticates with the controller
    Then the controller reports an authentication error
    And the SDK surfaces that error to the caller instead of a generic transport failure

  Scenario: Reusing a cached access token across multiple requests
    Given a client configured with the controller's client ID and client secret
    When the client makes 2 authenticated requests to list sites
    Then the controller's token endpoint was called exactly 1 time

  Scenario: Serializing concurrent token fetches into a single authorization request
    Given a client configured with the controller's client ID and client secret
    When 10 goroutines concurrently request an access token from the same provider
    Then the controller's token endpoint was called exactly 1 time
    And every goroutine received the same access token

  Scenario: Failing fast when configured with an empty access token
    Given a client configured with an empty access token
    When the client makes an authenticated request to list sites
    Then the SDK reports an empty access token error without contacting the controller
