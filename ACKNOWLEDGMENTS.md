# Acknowledgments

This project's testing patterns for the Azure DevOps integration were
inspired by the following open-source projects. No code was directly copied;
only architectural patterns and testing approaches were adopted.

## FluxCD notification-controller

- **Repository:** https://github.com/fluxcd/notification-controller
- **License:** Apache-2.0
- **What we learned:** Thin interface boundary pattern — define a minimal
  interface (only the methods you call) at the consuming site, inject via
  exported struct field, and use hand-written fakes (no mocking library) for
  unit tests. Integration tests gated behind `//go:build integration` with
  real infrastructure.

## mcdafydd/go-azuredevops

- **Repository:** https://github.com/mcdafydd/go-azuredevops
- **License:** BSD-3-Clause
- **What we learned:** The `httptest.NewServer` + `setup()` pattern
  (originated in google/go-github) for testing HTTP API clients. Testdata
  fixtures for webhook event payloads. Table-driven tests with shared
  assertion helpers.

## google/go-github

- **Repository:** https://github.com/google/go-github
- **License:** BSD-3-Clause
- **What we learned:** The canonical Go pattern for testing HTTP API clients:
  `setup()` returns `(client, mux, serverURL, teardown)`, handlers verify
  request shape and return canned JSON, assertions use `google/go-cmp`.

## Azure DevOps REST API Documentation

- **URL:** https://learn.microsoft.com/en-us/rest/api/azure/devops/?view=azure-devops-rest-7.2
- **Used for:** Response schema definitions in testdata fixtures and contract
  test struct definitions.
