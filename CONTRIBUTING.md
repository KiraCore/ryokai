## Project Workflow Guidelines

Our project adheres to the GitFlow workflow to enhance development and release management efficiency. This guide delineates our branching strategy and workflow steps, ensuring contributors can navigate and contribute effectively.

### Branch Types

- **main**: Houses stable production releases.
- **feature/***: For maining new features or enhancements, e.g., `feature/add-login`.
- **release/***: Manages pre-release activities and final testing, adhering to [Semantic Versioning (semver)](https://semver.org/ "Semantic Versioning 2.0.0").
- **hotfix/***: Addresses critical issues in production, e.g., `hotfix/login-issue`.
- **bugfix/***: Addresses minor issues in production like typo, non-critical errors etc.
### Workflow Overview

#### Setting Up for Development

- To ensure you work with the most current version of the `main` branch, especially for new features, it's advisable to clone the repository afresh. This can be achieved by cloning the `main` branch directly and limiting the clone to the most recent commits:

    ```bash
    git clone --branch main --single-branch --depth 10 https://github.com/KiraCore/ryokai.git
    ```

    This command clones the `main` branch, fetching only the last 10 commits to keep the clone lightweight and focused.

- After cloning, create a new branch from `main` for your feature work:

    ```bash
    cd ryokai
    git checkout -b feature/your-feature-name
    ```

#### Committing Your Changes

- Commit your changes using structured messages to enhance clarity and facilitate automated changelog generation. Our commit message format includes a type, a scope, a subject, an optional body, and an optional footer.

    **Example Commit Message**:

    ```bash
    git commit -S -m "feat(login): Implement OAuth login flow" \
    -m "Implemented the OAuth login flow using the XYZ library, enhancing security and efficiency." \
    -m "Resolves: #123, Related: #456"
    ```

    This format ensures commit clarity and aids in automated processes like changelog generation.

```plaintext
<type>(<scope>): <subject>

<body>

<footer>
```
        * Type: The nature of the change feat, fix, docs, refactor, test.
        * Scope: The scope of the change, such as feature, module, package, utils etc. Example: `feat(your-feature-name): Add feature`
        * Subject: A succinct description of the change, written in the imperative mood. Example: `Add...`, `Fix...`, `Replace...`
        * Body: A more detailed explanation of the change (optional).
        * Footer: References to related issues or pull requests (optional).


#### Updating and Finalizing Your Feature Branch

- Keep your branch updated with `main` to minimize conflicts, and consider squashing your commits for a cleaner history before merging.

    **Squashing Commits**:

    ```bash
    git rebase -i HEAD~N # N is the number of commits you want to squash
    ```

- Push your feature branch and create a Pull Request to `main`. Ensure the PR description is detailed, facilitating review.

    **PR Requirements**:

    - After all checks have passed, a PR will require at least 2 approvals from team members, ensuring thorough review and quality.

### Updating the Changelog

Maintaining an accurate and up-to-date changelog is essential for our project. It helps users and contributors understand the changes made between each release and can be crucial for diagnosing issues or understanding new features.

#### Changelog Format

Our project follows the [Keep a Changelog](https://keepachangelog.com/) format, which organizes changes under the following headings:

- **Added** for new features.
- **Changed** for changes in existing functionality.
- **Deprecated** for soon-to-be removed features.
- **Removed** for now removed features.
- **Fixed** for any bug fixes.
- **Security** in case of vulnerabilities.

#### When to Update the Changelog

You should update the changelog as part of your pull request whenever you:

- Introduce new features or enhancements.
- Fix bugs or issues within the project.
- Make significant changes that affect users or developers.

#### How to Update the Changelog

1. **Identify the Correct Section**: Based on the nature of your contribution, identify under which heading your update belongs.

2. **Write a Concise Description**: Provide a short and clear description of the change. Include issue or pull request numbers if applicable.

3. **Follow the Style**: Look at previous entries in the changelog for style guidance. Consistency helps readers quickly find the information they need.

### Example Entry

```markdown
## [Unreleased]

### Added
- OAuth2 login support for additional providers (GitHub, Google) to improve user experience. See PR #123.

### Fixed
- Resolved an issue where the application could crash under specific conditions when processing requests. Fixes issue #456.
```

