Core building blocks for any WSO2 surface.

```jsx
<IconButton aria-label="Settings"><i data-lucide="settings" /></IconButton>
<Badge tone="success" dot>Active</Badge>
<Badge tone="brand">Beta</Badge>
<Tag onRemove={() => {}}>region: us-east</Tag>
<Avatar name="Asha Perera" size="md" />
<Card interactive><h4>API Manager</h4><p>Full lifecycle API management.</p></Card>
```

- **IconButton** — square icon-only; `primary`/`secondary`/`ghost`. Always pass `aria-label`.
- **Badge** — status/category pill; tones `neutral`/`brand`/`success`/`warning`/`danger`/`info`/`solid`; optional `dot`.
- **Tag** — outlined chip for filters/metadata; `onRemove` adds a × button; optional `icon`.
- **Avatar** — image or colored-initials fallback; `sm`/`md`/`lg`.
- **Card** — white surface, subtle border + soft shadow, 16px radius; `interactive` lifts on hover.
