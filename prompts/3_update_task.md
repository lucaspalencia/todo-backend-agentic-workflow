Implement the "update task" feature.

Requirements:
- PATCH /tasks/{id} endpoint that accepts partial updates (any combination of title, description, status)
- Reuse existing validation rules from the create flow
- Follow the established DDD layer structure — add an update use case in the application layer
- Return 404 with a JSON error if the task does not exist
- Return 200 with the full updated task on success
- updated_at must be refreshed on every successful update
- Integration test covering: successful update, partial update (single field), not found, and validation errors
