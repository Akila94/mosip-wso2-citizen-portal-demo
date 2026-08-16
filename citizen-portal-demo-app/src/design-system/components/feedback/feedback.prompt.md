Feedback primitives.

```jsx
<Alert tone="success" title="Deployed" onClose={dismiss}>
  Your API revision is now live in us-east.
</Alert>
<Tooltip label="Copy endpoint URL"><IconButton aria-label="Copy"><i data-lucide="copy" /></IconButton></Tooltip>
```

- **Alert** — inline callout; tones `info`/`success`/`warning`/`danger`; optional `title` + `onClose`.
- **Tooltip** — hover/focus label on a wrapped child; `placement` top/bottom/left/right; dark on light.
