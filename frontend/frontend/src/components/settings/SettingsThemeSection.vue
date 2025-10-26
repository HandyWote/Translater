<script lang="ts" setup>
import {useSettingsForm} from './useSettingsForm';

const form = useSettingsForm();

const themeOptions = [
	{label: '跟随系统', value: 'system'},
	{label: '浅色', value: 'light'},
	{label: '深色', value: 'dark'},
];
</script>

<template>
	<div class="settings-theme">
		<div class="settings-theme__options">
			<label v-for="option in themeOptions" :key="option.value" class="settings-theme__item">
				<input v-model="form.theme" type="radio" name="theme" :value="option.value" />
				<div>
					<strong>{{ option.label }}</strong>
					<span v-if="option.value === 'system'">自动随操作系统切换深浅模式。</span>
					<span v-else>固定使用{{ option.label }}界面风格。</span>
				</div>
			</label>
		</div>
	</div>
</template>

<style scoped>
.settings-theme {
	display: flex;
	flex-direction: column;
	gap: 0.9rem;
}

.settings-theme__options {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.settings-theme__item {
	display: flex;
	gap: 0.75rem;
	align-items: flex-start;
	padding: 0.7rem 0.85rem;
	border-radius: 12px;
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	cursor: pointer;
	transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.settings-theme__item input {
	margin-top: 0.25rem;
}

.settings-theme__item strong {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-theme__item span {
	display: block;
	margin-top: 0.2rem;
	color: var(--color-text-tertiary);
	font-size: 0.78rem;
	line-height: 1.35;
}
</style>
