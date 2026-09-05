import { FILTER_OPTIONS, type FilterStatus } from '../types/filters.ts'
import styles from './FilterChips.module.css'

interface FilterChipsProps {
  selectedFilter: FilterStatus
  onSelectFilter: (filter: FilterStatus) => void
}

export function FilterChips({ selectedFilter, onSelectFilter }: FilterChipsProps) {
  return (
    <nav aria-label="Statusfilter" className={styles.container}>
      {FILTER_OPTIONS.map((option) => {
        const isActive = option === selectedFilter
        return (
          <button
            key={option}
            type="button"
            className={`${styles.chip} ${isActive ? styles.chipActive : ''}`}
            onClick={() => onSelectFilter(option)}
            aria-pressed={isActive}
          >
            {option}
          </button>
        )
      })}
    </nav>
  )
}
