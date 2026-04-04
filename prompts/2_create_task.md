Implement the "create task" feature.

Task model:
- id: UUID, generated server-side
- title: string, required, max 255 characters
- description: string, optional, max 2000 characters
- status: string, defaults to "pending" (allowed values: "pending", "in_progress", "done")
- created_at: timestamp
- updated_at: timestamp

Requirements:
- SQL migration to create the tasks table
- POST /tasks endpoint that accepts JSON body with title, description, and optional status
- Input validation with descriptive error messages returned as JSON
- Follow the project's DDD architecture:
  - Task entity and repository interface in the domain layer
  - Create task use case in the application layer
  - PostgreSQL repository implementation and HTTP handler in the infrastructure layer
- Return 201 with the created task on success
- Integration test covering: successful creation, validation errors, and duplicate handling
