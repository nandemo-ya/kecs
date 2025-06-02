# ListServices API

## Overview

The ListServices API returns a list of services running in the specified Amazon ECS cluster. You can filter the results by launch type and use pagination to retrieve large result sets.

## API Specification

### Request

```json
{
  "cluster": "string",
  "nextToken": "string",
  "maxResults": number,
  "launchType": "string",
  "schedulingStrategy": "string"
}
```

#### Parameters

- **cluster** (string, optional): The short name or full Amazon Resource Name (ARN) of the cluster to use when filtering the ListServices results. If you do not specify a cluster, the default cluster is assumed.
- **nextToken** (string, optional): The nextToken value returned from a previous paginated ListServices request where maxResults was used and the results exceeded the value of that parameter.
- **maxResults** (number, optional): The maximum number of service results returned by ListServices in paginated output. The default value is 100. The maximum value is 100.
- **launchType** (string, optional): The launch type to use when filtering the ListServices results. Valid values: EC2, FARGATE, EXTERNAL.
- **schedulingStrategy** (string, optional): The scheduling strategy to use when filtering the ListServices results. Valid values: REPLICA, DAEMON. (Note: Currently accepted but not used for filtering)

### Response

```json
{
  "serviceArns": [
    "string"
  ],
  "nextToken": "string"
}
```

#### Response Fields

- **serviceArns** (array): The list of full ARN entries for each service that's associated with the specified cluster.
- **nextToken** (string): The nextToken value to include in a future ListServices request. When the results of a ListServices request exceed maxResults, this value can be used to retrieve the next page of results.

## Behavior

1. **Default Behavior**:
   - Returns all services in the specified cluster
   - Default limit is 100 services per request
   - Results are ordered by service ID for consistent pagination

2. **Filtering**:
   - **Launch Type**: Filters services by their launch type (EC2, FARGATE, EXTERNAL)
   - **Scheduling Strategy**: Currently accepted but not applied as a filter

3. **Pagination**:
   - Use `maxResults` to limit the number of services returned
   - If more services exist, a `nextToken` is returned
   - Use the `nextToken` in subsequent requests to get the next page

## Error Conditions

- Returns an empty list if the cluster doesn't exist or has no services
- Invalid parameters result in appropriate error responses

## Example Usage

### List All Services
```bash
curl -X POST http://localhost:8080/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster"
  }'
```

### List Services with Pagination
```bash
# First page
curl -X POST http://localhost:8080/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "maxResults": 10
  }'

# Next page (using nextToken from previous response)
curl -X POST http://localhost:8080/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "maxResults": 10,
    "nextToken": "service-id-from-previous-response"
  }'
```

### Filter by Launch Type
```bash
# List only FARGATE services
curl -X POST http://localhost:8080/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "launchType": "FARGATE"
  }'

# List only EC2 services
curl -X POST http://localhost:8080/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "launchType": "EC2"
  }'
```

## Implementation Notes

- The service ARNs are returned in the format: `arn:aws:ecs:{region}:{accountId}:service/{cluster-name}/{service-name}`
- The pagination uses cursor-based pagination with the service ID as the cursor
- Empty result sets return an empty `serviceArns` array with no `nextToken`