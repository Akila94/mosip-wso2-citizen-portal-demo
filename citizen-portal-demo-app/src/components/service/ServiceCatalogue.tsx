import React from 'react';
import { ServiceCard } from './ServiceCard';
import type { ServiceCategory } from '../../services/types';
import styles from '../../styles/layout.module.css';

export function ServiceCatalogue({ categories, signedIn, onSelectService }: { categories: ServiceCategory[]; signedIn: boolean; onSelectService?: (serviceId: string) => void }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-8)' }}>
      {categories.map((category) => (
        <section key={category.id} aria-labelledby={`cat-${category.id}`}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)', borderBottom: '1.5px solid var(--border-default)', paddingBottom: 'var(--space-2)', marginBottom: 'var(--space-4)' }}>
            <h2 id={`cat-${category.id}`} style={{ margin: 0, font: '600 11px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase' }}>
              {category.name}
            </h2>
            <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-muted)' }}>{category.count}</span>
          </div>
          <ul className={styles.cardGrid} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
            {category.services.map((service) => (
              <li key={service.id}>
                <ServiceCard service={service} signedIn={signedIn} onClick={onSelectService ? () => onSelectService(service.id) : undefined} />
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
