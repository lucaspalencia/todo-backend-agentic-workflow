Implement the "delete task" feature.

Requirements:
- DELETE /tasks/{id} endpoint
- Return 204 (no content) on successful deletion
- Return 404 with a JSON error if the task does not exist
- Soft delete over hard delete
- Follow the established DDD layer structure for delete, list, and get-by-id use cases
- Integration test covering: successful deletion, not found, and verifying the task is actually removed

Also add a GET /tasks endpoint that returns all non deleted tasks as a JSON array (empty array if none exist), and a GET /tasks/{id} endpoint that returns a single task or 404 if task deleted. Add integration tests for both.
