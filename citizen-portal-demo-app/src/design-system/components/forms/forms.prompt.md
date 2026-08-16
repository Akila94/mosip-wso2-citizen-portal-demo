Form controls matching the WSO2 input system — 8px radius, orange focus ring, sentence-case labels.

```jsx
<Input label="Work email" placeholder="you@company.com" iconLeft={<i data-lucide="mail" />} />
<Input label="API name" error="This name is taken" />
<Select label="Region" options={["us-east", "eu-west", "ap-south"]} />
<Checkbox label="I agree to the terms" defaultChecked />
<Switch label="Enable rate limiting" defaultChecked />
```

- **Input** — `label`, `hint`, `error`, `iconLeft`, sizes `sm`/`md`/`lg`. Focus = orange ring.
- **Select** — styled native select; `options` are strings or `{value,label}`.
- **Checkbox** / **Switch** — controlled or uncontrolled; checked state uses signature orange.
