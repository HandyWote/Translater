<script lang="ts" setup>
import {computed, unref} from 'vue';
import type {Ref} from 'vue';
import type {SettingsCategory, CategoryKey} from '../../config/settings';

const props = defineProps<{
	categories: SettingsCategory[];
	activeCategory: CategoryKey | Ref<CategoryKey>;
	validationError: string | null;
}>();

const emit = defineEmits<{
	(event: 'select', value: CategoryKey): void;
}>();

const activeCategory = computed(() => unref(props.activeCategory));

function handleSelect(key: CategoryKey) {
	emit('select', key);
}
</script>

<template>
	<aside class="settings-nav">
		<div class="settings-nav__title">
			<h1>设置中心</h1>
		</div>
		<div class="settings-nav__header">
			<strong>配置分组</strong>
		</div>
		<nav class="settings-nav__list">
			<button
				v-for="category in props.categories"
				:key="category.key"
				type="button"
				class="settings-nav__item"
				:class="{active: category.key === activeCategory}"
				@click="handleSelect(category.key)"
			>
				<span class="settings-nav__icon" aria-hidden="true">{{ category.icon }}</span>
				<div class="settings-nav__text">
					<span class="settings-nav__label">{{ category.label }}</span>
					<span class="settings-nav__desc">{{ category.description }}</span>
				</div>
			</button>
		</nav>
		<div class="settings-nav__footer">
			<span v-if="props.validationError" class="settings-nav__alert">{{ props.validationError }}</span>
			<slot name="actions" />
		</div>
	</aside>
</template>

<style scoped>
.settings-nav {
	position: sticky;
	top: 92px;
	display: flex;
	flex-direction: column;
	gap: 1.25rem;
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-subtle);
	padding: 1.5rem 1.25rem;
	box-shadow: 0 14px 28px rgba(8, 12, 32, 0.14);
	min-width: 240px;
}

.settings-nav__title h1 {
	margin: 0;
	font-size: 1.15rem;
	font-weight: 600;
}

.settings-nav__header {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}

.settings-nav__header strong {
	font-size: 0.94rem;
	font-weight: 600;
}

.settings-nav__list {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
}

.settings-nav__item {
	display: flex;
	align-items: flex-start;
	gap: 0.85rem;
	width: 100%;
	text-align: left;
	padding: 0.75rem 0.95rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: transparent;
	color: inherit;
	cursor: pointer;
	transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.settings-nav__item:hover {
	border-color: var(--border-subtle);
	background: var(--surface-hover);
}

.settings-nav__item.active {
	border-color: var(--accent);
	background: rgba(20, 131, 255, 0.12);
	color: var(--accent);
	box-shadow: 0 10px 22px rgba(20, 131, 255, 0.18);
}

.settings-nav__icon {
	font-size: 1.2rem;
	line-height: 1;
}

.settings-nav__text {
	display: flex;
	flex-direction: column;
	gap: 0.18rem;
}

.settings-nav__label {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-nav__desc {
	font-size: 0.78rem;
	color: var(--color-text-tertiary);
	line-height: 1.35;
}

.settings-nav__footer {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.settings-nav__alert {
	display: inline-flex;
	align-items: center;
	gap: 0.4rem;
	padding: 0.45rem 0.8rem;
	border-radius: 999px;
	background: rgba(255, 86, 48, 0.12);
	border: 1px solid rgba(255, 86, 48, 0.28);
	color: #d03a16;
	font-size: 0.82rem;
	line-height: 1.2;
}
</style>
