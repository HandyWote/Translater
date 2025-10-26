<script lang="ts" setup>
import {computed, unref} from 'vue';
import type {Ref} from 'vue';

const props = defineProps<{
	variant?: 'primary' | 'ghost' | 'toolbar';
	type?: 'button' | 'submit' | 'reset';
	loading?: boolean | Ref<boolean>;
	disabled?: boolean | Ref<boolean>;
}>();

const variantClass = computed(() => {
	switch (props.variant) {
	case 'ghost':
		return 'app-button--ghost';
	case 'toolbar':
		return 'app-button--toolbar';
	default:
		return 'app-button--primary';
	}
});

const isLoading = computed(() => Boolean(unref(props.loading)));
const isDisabled = computed(() => Boolean(unref(props.disabled) || isLoading.value));
</script>

<template>
	<button
		:class="['app-button', variantClass, {loading: isLoading}]"
		:type="props.type ?? 'button'"
		:disabled="isDisabled"
	>
		<span v-if="isLoading" class="app-button__spinner" aria-hidden="true"></span>
		<span class="app-button__content">
			<slot />
		</span>
	</button>
</template>

<style scoped>
.app-button {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: 0.35rem;
	border: none;
	border-radius: 999px;
	padding: 0.6rem 1.4rem;
	font-size: 0.95rem;
	font-weight: 500;
	cursor: pointer;
	transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease, border-color 0.15s ease;
	position: relative;
}

.app-button:disabled {
	opacity: 0.52;
	cursor: not-allowed;
	box-shadow: none;
}

.app-button--primary {
	background: var(--accent);
	color: #fff;
	box-shadow: 0 10px 18px rgba(20, 131, 255, 0.25);
}

.app-button--primary:not(:disabled):hover {
	transform: translateY(-1px);
}

.app-button--ghost {
	background: transparent;
	color: var(--color-text-secondary);
	border: 1px solid var(--border-subtle);
}

.app-button--ghost:not(:disabled):hover {
	background: var(--surface-hover);
}

.app-button--toolbar {
	background: transparent;
	border: 1px solid transparent;
	color: inherit;
	padding: 0.45rem 1rem;
}

.app-button--toolbar:not(:disabled):hover {
	background: var(--surface-hover);
}

.app-button__spinner {
	margin-right: 0.2rem;
	width: 16px;
	height: 16px;
	border: 3px solid rgba(255, 255, 255, 0.3);
	border-top-color: #fff;
	border-radius: 50%;
	animation: app-button-spin 0.8s linear infinite;
}

.app-button--ghost .app-button__spinner {
	border-color: rgba(66, 133, 244, 0.3);
	border-top-color: var(--accent);
}

.app-button__content {
	display: inline-flex;
	align-items: center;
	gap: 0.35rem;
}

@keyframes app-button-spin {
	to {
		transform: rotate(360deg);
	}
}
</style>
