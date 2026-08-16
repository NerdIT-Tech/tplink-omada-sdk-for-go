Feature: Listing managed sites
  As an integrator building on top of the Omada SDK
  I want to list the sites a controller manages
  So that my application can let a user pick which site to operate on

  Background:
    Given an authenticated client for the controller

  Scenario: Listing the first page of sites
    When the client requests page 1 of sites with a page size of 10
    Then the response envelope reports success
    And the response contains a page of site summaries

  Scenario: Listing a page beyond the available data returns an empty, successful page
    When the client requests page 9999 of sites with a page size of 10
    Then the response envelope reports success
    And the response contains an empty page of site summaries

  Scenario: Rejecting an out-of-range page number through the response envelope
    When the client requests page 0 of sites with a page size of 10
    Then the response envelope reports a controller-side validation error

  Scenario: Rejecting an out-of-range page size through the response envelope
    When the client requests page 1 of sites with a page size of 0
    Then the response envelope reports a controller-side validation error

  Scenario: Rejecting requests for an unrecognized Omada Cloud Controller ID
    When the client requests sites using an unrecognized Omada Cloud Controller ID
    Then the response envelope reports a controller-side validation error
