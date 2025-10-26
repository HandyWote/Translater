<script lang="ts" setup>
const props = withDefaults(
	defineProps<{
		maxWidth?: number;
		padded?: boolean;
	}>(),
	{maxWidth: 960, padded: true},
);
</script>

<template>
	<section class="panel-shell">
		<div :class="['panel-shell__inner', {padded: props.padded}]" :style="{maxWidth: `${props.maxWidth}px`}">
			<slot />
		</div>
	</section>
</template>

<style scoped>
.panel-shell {
	display: flex;
	justify-content: center;
	width: 100%;
}

.panel-shell__inner {
	width: 100%;
	display: flex;
	flex-direction: column;
	gap: 1.2rem;
}

.panel-shell__inner.padded {
	padding: 0;
}

@media (max-width: 1000px) {
	.panel-shell__inner {
		max-width: 100% !important;
	}
}
</style>
