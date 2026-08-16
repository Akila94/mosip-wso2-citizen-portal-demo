The WSO2 wordmark — use anywhere the brand needs to appear (nav, footer, login, deck). Lowercase, Inter Extrabold, with the "2" in signature orange.

```jsx
<Logo size={28} />
<Logo size={24} tone="light" />        {/* on dark surfaces */}
<Logo size={40} tone="mono" accent={false} /> {/* single-color */}
```

Variants: `tone="auto"` (dark text), `"light"` (white), `"mono"` (single color, no orange). `accent` toggles the orange "2"; `showAccentDot` adds a trailing orange dot. Note: type-recreated from brand fonts — replace with official logo art when exact fidelity is needed.
