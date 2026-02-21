# Wiki Update: Container Runner Abstraction

After the container runner abstraction refactor, the **huskyCI.wiki** repo should be updated so documentation matches the new architecture. Apply these updates in `~/Gits/huskyCI.wiki` (or wherever the wiki repo is cloned).

## Summary of Code Changes

- The API no longer talks to "a Docker host" directly from many places. It uses a **Runner** interface (`api/runner`).
- **Docker runner** (default): Uses existing `HUSKYCI_DOCKERAPI_ADDR` and `HUSKYCI_DOCKERAPI_PORT` to build the Docker host URL. Can target Docker-in-Docker (DinD) or the host daemon (e.g. `host.docker.internal` or Unix socket).
- **Optional (future):** `HUSKYCI_RUNNER_TYPE=remote` and `HUSKYCI_RUNNER_ADDR` for a remote runner service; the API would then send HTTP requests instead of using a Docker client.

## Wiki Pages to Update

1. **Architecture / Installation**
   - Describe the **container runner** abstraction: the API requests container execution (security tests, zip extraction) through a single interface.
   - Deployment options:
     - **Docker (default):** Runner talks to a Docker API (TCP or Unix). Can be DinD (e.g. `dockerapi:2376`) or host Docker (e.g. `host.docker.internal:2375` or `unix:///var/run/docker.sock`).
     - **Remote runner (optional):** API points at a separate runner service via HTTP; the service runs on the host with Docker socket.
   - Replace any wording that assumes "the API always talks to DinD" with "the API uses a configurable runner (Docker by default)."

2. **Environment Variables**
   - **Existing (unchanged):** `HUSKYCI_DOCKERAPI_ADDR`, `HUSKYCI_DOCKERAPI_PORT`, `HUSKYCI_DOCKERAPI_TLS_VERIFY`, `HUSKYCI_DOCKERAPI_CERT_PATH` — used when the runner type is Docker (default).
   - **Optional (when implemented):** `HUSKYCI_RUNNER_TYPE` (e.g. `docker` | `remote`), `HUSKYCI_RUNNER_ADDR` (base URL for remote runner service).
   - Note: No new required env vars for current deployments; default remains Docker runner with existing Docker host config.

3. **Configuration / Deployment Guides**
   - Any page that explains "Docker API" or "DinD" setup: clarify that the API uses a **runner** that, in the default setup, connects to the Docker API at the configured address (DinD in Compose, or host if so configured).
   - Local deployment: reference `LOCAL_DEPLOYMENT.md` in the main repo for runner abstraction and host vs DinD options.

4. **Troubleshooting**
   - "Runner unavailable" or "no container runner configured": ensure `HUSKYCI_INFRASTRUCTURE_USE=docker` and Docker (or runner) health check succeeds at startup.
   - Docker-specific errors: same as before (check `HUSKYCI_DOCKERAPI_*` and that the daemon is reachable from the API).

## Checklist

- [ ] Architecture/overview page updated with runner abstraction and deployment options.
- [ ] Env vars page lists Docker host vars and notes optional runner type/address for future.
- [ ] Installation/deployment pages no longer imply DinD-only; mention host Docker option where relevant.
- [ ] Troubleshooting mentions runner and points to Docker/runner config.
