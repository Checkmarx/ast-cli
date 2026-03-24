# Custom Constitution Principle Snippets

This folder contains markdown snippets injected into `constitution.md` when their registry detection matches.

## How to add a snippet

1. Create a new markdown file in this folder.
2. Add a heading prefix in the first heading:

```md
### [CP:your_principle_id] Principle Title
```

3. Add a clear section title and principle content.
4. Register it in:

`../custom-constitution-principles.json`

## Important

- `CP:your_principle_id` should match the `id` in the registry.
- Use explicit `MUST` / `MUST NOT` statements only for truly non-negotiable constraints.
