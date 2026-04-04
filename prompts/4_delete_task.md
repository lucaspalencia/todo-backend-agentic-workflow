Implement the "delete task" feature.

Requirements:
- DELETE /tasks/{id} endpoint
- Return 204 (no content) on successful deletion
- Return 404 with a JSON error if the task does not exist
- Soft delete is not required — a hard delete is fine
- Follow the established DDD layer structure for delete, list, and get-by-id use cases
- Integration test covering: successful deletion, not found, and verifying the task is actually removed (GET after DELETE should 404)

Also add a GET /tasks endpoint that returns all tasks as a JSON array (empty array if none exist), and a GET /tasks/{id} endpoint that returns a single task or 404. Add integration tests for both.
