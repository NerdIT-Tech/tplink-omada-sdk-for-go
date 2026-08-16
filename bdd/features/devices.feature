Feature: Listing site devices
  As an integrator building on top of the Omada SDK
  I want to list and search the devices adopted at a site
  So that my application can show a user their network hardware

  Background:
    Given an authenticated client for the controller
    And the controller's first managed site

  Scenario: Listing devices for a site
    When the client requests page 1 of devices with a page size of 50
    Then the response envelope reports success
    And the response contains at least 1 device

  Scenario: Filtering devices by a search key
    When the client requests devices with a page size of 50 matching search key "wap"
    Then the response envelope reports success
    And every device in the response has "wap" in its name

  Scenario: Sorting devices by name ascending
    When the client requests devices with a page size of 50 sorted by name ascending
    Then the response envelope reports success
    And the devices in the response are sorted by name ascending

  Scenario: Searching for a device that does not exist returns an empty, successful page
    When the client requests devices with a page size of 50 matching search key "zzz-does-not-exist"
    Then the response envelope reports success
    And the response contains an empty page of devices

  # Documents a known SDK limitation rather than desired behavior: the generated
  # V1ItemSitesItemDevicesRequestBuilder.Get() takes an *optional* QueryParameters
  # struct, but the controller's URL template treats page/pageSize as required. Omit
  # them and the controller responds with a plain "Bad Request" page instead of its
  # usual {errorCode,msg} envelope, so the SDK can only surface a generic status-code
  # error. See QA_REPORT.md for the recommendation.
  @known-limitation
  Scenario: Omitting required pagination parameters yields an opaque error
    When the client requests devices without specifying a page or page size
    Then the SDK reports a generic error instead of the controller's specific reason
