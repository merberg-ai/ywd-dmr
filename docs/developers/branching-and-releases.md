# Branching and Releases

YWD-DMR uses a simple two-branch workflow so development can move quickly without turning the normal install/update path into a moving target.

## `main` — tested/stable branch

`main` is the branch for code that has passed the project's current test and review expectations.

Normal users, release packages, and the stable update channel should ultimately be based on `main`.

Do not use `main` as the everyday development branch. A feature belongs on `main` only after it has been developed on `dev`, tested, and intentionally promoted.

## `dev` — active development branch

`dev` is the working branch for ongoing YWD-DMR development.

New features, frontend changes, API work, installer work, documentation changes, experimental support, and fixes should normally land on `dev` first.

Development builds may be incomplete or broken. They are intended for developers and testers, not ordinary appliance users.

## Normal workflow

1. Start from the current `dev` branch.
2. Make the code, WebUI, installer, protocol, or documentation change.
3. Update `/docs/` in the same change whenever behavior or operation changed.
4. Run tests, vet, native build, and the Pi Zero / ARMv6 build.
5. Test the resulting development build on appropriate real hardware when the change needs hardware verification.
6. When a development snapshot is considered good, promote `dev` to `main` through a reviewed merge/pull request.
7. Create releases/tags from the tested `main` state.

## Release/update channels

The planned appliance updater will eventually expose channels such as:

- **Stable** — tagged releases based on tested `main` commits.
- **Development** — opt-in builds/releases based on `dev` for testers.

Stable must remain the normal-user default once stable releases exist.

## Hotfixes

If a serious problem is found in a released build, make the fix in a way that keeps `dev` and `main` from drifting permanently. The fix must end up in both development history and the tested release line.

## Documentation rule

Branching, release, install, and update behavior is part of the product. If this workflow changes, update this document and any affected installer/update documentation in the same change.
