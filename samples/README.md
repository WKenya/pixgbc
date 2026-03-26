# Samples

Deterministic sample inputs for manual QA, docs, and visual regression review.

Files:

- `gradient-landscape.png`: relaxed-mode baseline
- `portrait-alpha.png`: alpha flattening baseline
- `tile-banks.png`: strict `cgb-bg` palette-bank baseline

Regenerate:

```sh
make samples
make sample-outputs
```

Derived artifacts land in:

- `samples/outputs/`
- `samples/reviews/`

Those derived directories are ignored in git. Keep source inputs + generator in version control; regenerate outputs/reviews locally as needed.
