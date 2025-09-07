# zygarde
Zygarde is a modern, modular tool for environment setup and deployment. Embracing the philosophy of Pokémon's Zygarde, it is dedicated to maintaining "order" and "integrity" in development environments.


Building a declarative, developer-friendly tool for one-click deployment of local database environments.
Users define their desired database cluster topology through a simple configuration file, and the tool automatically generates and executes standard container orchestration configurations, thereby shielding the underlying technical complexity.

Container Orchestration Configuration (Phase 1: Docker Compose, Phase 2: K8s)

Core Modules:
### Template Manager
Template CRUD: Handles uploading (Create), reading (Read), updating (Update), deleting (Delete), and listing (List) of templates.
Template Parsing and Validation: When uploading a template, it parses the content, extracts defined variables (e.g., from {{ .Port }} or metadata), and validates the template syntax and variable definitions.
Template Information Provision: Provides template content and variable specifications to the "Blueprint Manager" for orchestration and variable assignment.

Template Manager: The warehouse keeper managing "parts" (templates).

### Blueprint Manager

Blueprint CRUD: Handles the creation, reading, updating, deletion, and listing of blueprints.
Blueprint Orchestration: Manages which "templates" constitute a "blueprint."
Variable Management: Manages the specific variable values used by each referenced template within the blueprint.
Blueprint Rendering (Core): Fetches the required templates based on the blueprint definition, injects variable values, and combines all template fragments into a complete, final docker-compose.yaml file.

Blueprint Manager: The designer managing "blueprints" (blueprints), assembling parts into products based on the blueprints.

### Environment Manager
Environment CRUD and State Management: Handles the creation, reading, updating, and deletion of environment instances, and persistently records the state of each environment (e.g., Creating, Running, Stopped, Error).
Environment-Blueprint Association: Records which "blueprint" created each environment instance.
Metadata Management: Manages environment metadata such as unique ID, name, creation time, access endpoints (e.g., generated IP and port).
Status Querying: Provides APIs for external queries about the current status and information of environments.

Environment Manager: The inventory manager overseeing "production lines" and "product instances" (environments), tracking the current status of each product.

### Deployment Engine
Executing Deployment Commands: Executes commands such as docker-compose up -d, down, stop, start, etc.
Project Isolation: Ensures that the docker-compose.yaml file generated for each "environment" runs as an independent Docker Compose project (via the -p parameter with a unique project name) to avoid naming conflicts.
Status Capture and Feedback: Captures command execution results (success, failure, output) and provides feedback to the caller.
Future Extensibility: Supports other orchestration platforms like Kubernetes.

Deployment Engine: The "robotic arm" on the assembly line, solely responsible for executing physical actions like "assemble," "pause," "start," and "destroy."

### Coordinator / Unified Facade
Process Orchestration: Acts as the brain of the system, connecting the functionalities of all above components to complete user instructions.
API Exposure: Provides a unified kernel interface for both CLI and Web API layers.
Error Handling and Transactional Guarantees: Coordinates calls across multiple components, performing cleanup and state rollbacks in case of errors (e.g., cleaning up via the Deployment Engine and updating the environment status to "Error" after a failed creation).

Coordinator: The "chief engineer" who receives orders (user commands), directs the warehouse keeper to find blueprints and parts, commands the robotic arm to work, and updates inventory status in real-time.
