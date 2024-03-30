# Kubebuilder Roadmaps

**Welcome to the Kubebuilder Roadmaps directory!**

This space is dedicated to housing the strategic roadmaps for the
Kubebuilder project, organized by year. Each document within this repository
outlines the key initiatives, objectives, and goals for Kubebuilder, reflecting our
commitment to enhancing the development experience within the Kubernetes ecosystem.

Below, you will find links to the roadmap document for each year. These documents provide insights into the
specific objectives set for the project during that time, the motivation behind each goal, and the progress
made towards achieving them:

- [Roadmap 2024](roadmap_2024.md)

## :point_right: New plugins/RFEs to provide integrations within other Projects

As Kubebuilder evolves, we prioritize a focused project scope and minimal reliance on third-party dependencies,
concentrating on features that bring the most value to our community.

While recognizing the need for flexibility, we opt not to directly support third-party project integrations.
Instead, we've enhanced Kubebuilder as a library, enabling any project to create compatible plugins.
This approach delegates maintenance to those with the deepest understanding of their projects, fostering higher
quality and community contributions.

We're here to support you in developing your own Kubebuilder plugins.
For guidance on [Creating Your own plugins](https://kubebuilder.io/plugins/creating-plugins).

This strategy empowers our users and contributors to innovate,
keeping Kubebuilder streamlined and focused on essential Kubernetes development functionalities.

**Therefore, our primary objective remains to offer a CLI tool that assists users in developing
solutions for deployment and distribution on Kubernetes clusters using Golang.
We aim to simplify the complexities involved and speed up the development process,
thereby lowering the learning curve.**

## :steam_locomotive: Contributing

Your input and contributions are what make Kubebuilder a continually
evolving and improving project. We encourage the community to participate in discussions,
provide feedback on the roadmaps, and contribute to the development efforts.

If you have suggestions for future objectives or want to get involved
in current initiatives, please refer to our [contributing guidelines](./../CONTRIBUTING.md)
or reach out to the project maintainers. Please, feel free either
to raise new issues and/or Pull Requests against this repository with your
suggestions.

## :loudspeaker: Stay Tuned

For the latest updates, discussions, and contributions to the Kubebuilder project,
please join our community channels and forums. Your involvement is crucial for the
sustained growth and success of Kubebuilder.

**:tada: Thank you for being a part of the Kubebuilder journey.**

Together, we are building the future of Kubernetes development.

## Template for roadmap items

```markdown
### [Goal Title]

**Status:** [Status Emoji] [Short Status Update]

**Objective:** [Brief description of the objective]

**Context:** [Optional - Any relevant background or broader context]

**Motivations:** [Optional - If applicable]
- [Key motivation 1]
- [Key motivation 2]

**Proposed Solutions:** [Optional - If applicable]
- [Solution 1]
- [Solution 2]
- [More as needed]

**References:** [Optional - Links to discussions, PRs, issues, etc.]
- [Reference 1 with URL]
- [Reference 2 with URL]
- [More as needed]
```