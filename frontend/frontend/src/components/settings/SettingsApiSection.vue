<script lang="ts" setup>
import {computed} from 'vue';
import {DEFAULT_API_BASE_URL} from '../../types';
import {useSettingsForm} from './useSettingsForm';

const props = defineProps<{
	showTranslateFields: boolean;
	apiKeyMissing: boolean;
}>();

const form = useSettingsForm();
</script>

<template>
	<div class="settings-grid">

		<div class="settings-grid__row">
			<label class="settings-field">
				<span>视觉 API Key</span>
				<input v-model="form.visionApiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off" />
				<small>默认用于视觉直出调用，如启用文本模型可再单独配置翻译 Key。</small>
			</label>
			<label v-if="props.showTranslateFields" class="settings-field">
				<span>翻译 API Key</span>
				<input v-model="form.apiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off" />
				<small>保存后即时生效，仅存储于当前用户配置目录。</small>
			</label>
		</div>
		<div class="settings-grid__row">
			<label class="settings-field">
				<span>视觉 API Base URL</span>
				<input v-model="form.visionApiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off" />
				<small>请输入兼容 OpenAI Chat Completions 的接口地址，结尾无需斜杠。</small>
			</label>
			<label v-if="props.showTranslateFields" class="settings-field">
				<span>翻译 API Base URL</span>
				<input v-model="form.apiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off" />
				<small>支持为翻译模型独立配置地址，留空时沿用视觉接口地址。</small>
			</label>
		</div>
	</div>
</template>

<style scoped>
.settings-helper {
	margin: 0 0 0.6rem;
	font-size: 0.85rem;
	color: var(--color-text-tertiary);
}

.settings-grid {
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
</style>
