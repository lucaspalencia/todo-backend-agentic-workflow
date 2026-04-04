Implement a "comments" feature for tasks.

Comment model:
- id: UUID, generated server-side
- task_id: UUID, foreign key referencing tasks, required
- content: string, required, max 2000 characters
- created_at: timestamp

Requirements:
- SQL migration to create the comments table with a foreign key constraint and ON DELETE CASCADE
- POST /tasks/{id}/comments endpoint to add a comment to a task (return 201)
- GET /tasks/{id}/comments endpoint to list all comments for a task, ordered by created_at ascending
- Return 404 if the parent task does not exist for both endpoints
- Validation with descriptive JSON error messages
- Follow the established DDD architecture — Comment entity and repository interface in domain, use cases in application, implementations in infrastructure
- Integration tests covering: adding a comment, listing comments, parent task not found, validation errors, and verifying cascade delete (deleting a task removes its comments)
