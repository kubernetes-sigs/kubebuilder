# Windows Support

Since no efforts have been made to add support for Windows in the past couple of years, we have decided not to pursue native Windows support at this time, considering both the additional maintenance overhead it adds to the project and the limited community contributions in that area.

That said, it’s still possible to use and contribute to Kubebuilder on a Windows machine by using WSL2 (Windows Subsystem for Linux) together with Docker Desktop.

- Learn more about setting up WSL2 in the [official docs](https://learn.microsoft.com/en-us/windows/wsl/).
- The [Docker Desktop documentation](https://docs.docker.com/desktop/features/wsl/) has instructions on how to set up Docker to use WSL2 as the backend on Windows.
- You can also learn more about setting up kind with Docker on WSL2 in the [kind official documentation](https://kind.sigs.k8s.io/docs/user/using-wsl2/).

All other dependencies and environment settings can be set up by following the Linux instructions.
