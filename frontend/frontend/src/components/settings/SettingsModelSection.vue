<script lang="ts" setup>
import {computed} from 'vue';
import {DEFAULT_TRANSLATE_MODEL, DEFAULT_VISION_MODEL, defaultSettingsState} from '../../types';
import {useSettingsForm} from './useSettingsForm';
import AppButton from '../base/AppButton.vue';

const form = useSettingsForm();

const showTranslateModelField = computed(() => !form.useVisionForTranslation);

function resetModels() {
	const defaults = defaultSettingsState();
	form.apiBaseUrl = defaults.apiBaseUrl;
	form.visionApiBaseUrl = defaults.visionApiBaseUrl;
	form.visionApiKeyOverride = defaults.visionApiKeyOverride;
	form.translateModel = defaults.translateModel;
	form.visionModel = defaults.visionModel;
}
</script>

<template>
	<div class="settings-models">
		<div class="settings-grid__row">
			<label class="settings-field">
				<span>视觉模型</span>
				<input v-model="form.visionModel" type="text" :placeholder="DEFAULT_VISION_MODEL" autocomplete="off" />
				<small>用于多模态识别与翻译直出。</small>
			</label>
			<label v-if="showTranslateModelField" class="settings-field">
				<span>翻译模型</span>
				<input v-model="form.translateModel" type="text" :placeholder="DEFAULT_TRANSLATE_MODEL" autocomplete="off" />
				<small>用于文本翻译（Chat Completions）。</small>
			</label>
		</div>
		<div class="settings-toggles">
			<label class="settings-toggle settings-toggle--primary">
				<input v-model="form.useVisionForTranslation" type="checkbox" />
				<div>
					<strong>视觉直出</strong>
					<span>直接使用视觉模型完成翻译，减少多轮调用。</span>
				</div>
			</label>
			<label class="settings-toggle">
				<input v-model="form.enableStreamOutput" type="checkbox" />
				<div>
					<strong>流式输出</strong>
					<span>逐段推送翻译结果，方便快速预览。</span>
				</div>
			</label>
		</div>
	</div>
</template>

<style scoped>
.settings-models {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.settings-grid__row {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
	gap: 1rem;
}

.settings-field {
	display: flex;
	flex-direction: column;
	gap: 0.45rem;
	font-size: 0.9rem;
}

.settings-field span {
	font-weight: 500;
}

.settings-field input {
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	border-radius: 12px;
	padding: 0.6rem 0.9rem;
	color: var(--color-text-primary);
	transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.settings-field input:focus {
	outline: none;
	border-color: var(--accent);
	box-shadow: 0 0 0 2px rgba(20, 131, 255, 0.25);
}

.settings-field small {
	color: var(--color-text-tertiary);
	font-size: 0.78rem;
	line-height: 1.4;
}

.settings-toggles {
	display: flex;
	flex-direction: column;
	gap: 0.8rem;
}

.settings-toggle {
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

.settings-toggle input {
	margin-top: 0.3rem;
}

.settings-toggle strong {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-toggle span {
	display: block;
	margin-top: 0.2rem;
	color: var(--color-text-tertiary);
	font-size: 0.78rem;
	line-height: 1.35;
}

.settings-toggle--primary {
	border-color: rgba(20, 131, 255, 0.22);
}

.settings-actions {
	display: flex;
	justify-content: flex-end;
}
</style>
