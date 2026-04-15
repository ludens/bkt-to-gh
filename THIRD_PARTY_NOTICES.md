# Third-Party Notices

This project is licensed under the MIT License. See [LICENSE](LICENSE).

Third-party dependency licenses were inspected with `go-licenses`:

```bash
go run github.com/google/go-licenses@latest report ./...
go run github.com/google/go-licenses@latest save ./... --save_path=third_party_licenses
```

Current dependency license summary:

- MIT
  - github.com/aymanbagabas/go-osc52/v2
  - github.com/charmbracelet/bubbletea
  - github.com/charmbracelet/lipgloss
  - github.com/charmbracelet/x/ansi
  - github.com/charmbracelet/x/term
  - github.com/lucasb-eyer/go-colorful
  - github.com/mattn/go-isatty
  - github.com/mattn/go-runewidth
  - github.com/muesli/ansi
  - github.com/muesli/cancelreader
  - github.com/muesli/termenv
  - github.com/rivo/uniseg
  - github.com/zalando/go-keyring
- BSD-2-Clause
  - github.com/godbus/dbus/v5
- BSD-3-Clause
  - golang.org/x/sync/errgroup
  - golang.org/x/sys/unix
  - golang.org/x/term

No GPL, LGPL, or AGPL dependencies were found in the current module graph.

If you need collected license texts for release or audit work, generate them
with:

```bash
go run github.com/google/go-licenses@latest save ./... --save_path=third_party_licenses
```
