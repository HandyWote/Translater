<script lang="ts" setup>
const props = defineProps<{
	title: string;
	description?: string;
	expanded: boolean;
}>();

const emit = defineEmits<{
	(event: 'toggle'): void;
}>();

function handleToggle() {
	emit('toggle');
}
</script>

<template>
	<section :class="['settings-card', {collapsed: !props.expanded}]">
		<header class="settings-card__header">
			<div class="settings-card__text">
				<h2>{{ props.title }}</h2>
				<p v-if="props.description">{{ props.description }}</p>
			</div>
			<div class="settings-card__header-actions">
				<slot name="header-actions" />
				<button type="button" class="settings-card__toggle" @click="handleToggle">
					{{ props.expanded ? '收起' : '展开' }}
				</button>
			</div>
		</header>
		<div class="settings-card__body" v-show="props.expanded">
			<slot />
		</div>
	</section>
</template>

<style scoped>
.settings-card {
	display: flex;
	flex-direction: column;
	gap: 1rem;
	background: var(--surface-elevated);
	border: 1px solid var(--border-subtle);
	border-radius: 18px;
	padding: 1.2rem 1.4rem;
	box-shadow: 0 16px 30px rgba(8, 12, 32, 0.16);
}

.settings-card.collapsed {
	box-shadow: none;
}

.settings-card__header {
	display: flex;
	gap: 1rem;
	justify-content: space-between;
	align-items: flex-start;
}

.settings-card__text h2 {
	margin: 0;
	font-size: 1.08rem;
	font-weight: 600;
}

.settings-card__text p {
	margin: 0.35rem 0 0;
	color: var(--color-text-tertiary);
	font-size: 0.85rem;
	line-height: 1.4;
}

.settings-card__header-actions {
	display: flex;
	align-items: center;
	gap: 0.6rem;
}

.settings-card__toggle {
	background: transparent;
	border: 1px solid var(--border-subtle);
	color: var(--color-text-secondary);
	border-radius: 999px;
	padding: 0.35rem 0.8rem;
	font-size: 0.82rem;
	cursor: pointer;
	transition: background 0.18s ease, border-color 0.18s ease;
}

.settings-card__toggle:hover {
	background: var(--surface-hover);
}

.settings-card__body {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}
</style>
