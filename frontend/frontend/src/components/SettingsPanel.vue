<script lang="ts" setup>
import {reactive, watch} from 'vue';
import type {SettingsState} from '../types';

const props = defineProps<{
	settings: SettingsState;
	apiKeyMissing: boolean;
}>();

const emit = defineEmits<{
	(event: 'submit', value: SettingsState): void;
}>();

const form = reactive<SettingsState>({...props.settings});

watch(
	() => props.settings,
	(next) => {
		Object.assign(form, next);
	},
	{deep: true},
);

function handleSubmit(event: Event) {
	event.preventDefault();
	emit('submit', {...form});
}

const languageOptions = [
	{label: '中文 (简体)', value: 'zh-CN'},
	{label: '中文 (繁体)', value: 'zh-TW'},
	{label: 'English', value: 'en-US'},
	{label: '日本語', value: 'ja-JP'},
	{label: '한국어', value: 'ko-KR'},
];

const themeOptions = [
	{label: '跟随系统', value: 'system'},
	{label: '浅色', value: 'light'},
	{label: '深色', value: 'dark'},
];
</script>

<template>
	<form class="settings" @submit="handleSubmit">
		<section class="card">
			<header>
				<h2>API 访问</h2>
				<p>{{ props.apiKeyMissing ? '未检测到 API Key，请输入有效凭证。' : '已配置 API Key，可直接使用截图和翻译功能。' }}</p>
			</header>
			<label class="field">
				<span>智谱 API Key</span>
				<input v-model="form.apiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off"/>
				<small>保存后立即生效，仅保存在本机用户配置目录。</small>
			</label>
		</section>

		<section class="card">
			<header>
				<h2>翻译行为</h2>
				<p>控制翻译语言、复制等自动化行为。</p>
			</header>
			<div class="grid">
				<label class="field">
					<span>目标语言</span>
					<select v-model="form.targetLanguage">
						<option v-for="option in languageOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
					</select>
				</label>
				<label class="toggle">
					<input v-model="form.autoCopyResult" type="checkbox"/>
					<div>
						<strong>翻译结果自动复制</strong>
						<span>完成翻译后自动写入剪贴板。</span>
					</div>
				</label>
				<label class="toggle">
					<input v-model="form.keepWindowOnTop" type="checkbox"/>
					<div>
						<strong>窗口置顶显示</strong>
						<span>前台保持窗口，方便对照阅读。</span>
					</div>
				</label>
				<label class="toggle">
					<input v-model="form.showToastOnComplete" type="checkbox"/>
					<div>
						<strong>显示完成提示</strong>
						<span>翻译完成时弹出底部提醒。</span>
					</div>
				</label>
			</div>
		</section>

		<section class="card">
			<header>
				<h2>界面主题</h2>
				<p>根据喜好切换深浅色或跟随系统。</p>
			</header>
			<label class="field">
				<span>界面主题</span>
				<select v-model="form.theme">
					<option v-for="option in themeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
				</select>
			</label>
		</section>

		<div class="actions">
			<button type="submit" class="primary">保存设置</button>
		</div>
	</form>
</template>

<style scoped>
.settings {
	display: flex;
	flex-direction: column;
	gap: 1.3rem;
	color: var(--color-text-primary);
}

.card {
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-subtle);
	padding: 1.3rem 1.5rem;
	box-shadow: 0 16px 28px rgba(10, 10, 30, 0.22);
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

header h2 {
	margin: 0;
	font-size: 1.1rem;
	font-weight: 600;
}

header p {
	margin: 0.35rem 0 0;
	font-size: 0.85rem;
	color: var(--color-text-tertiary);
}

.field {
	display: flex;
	flex-direction: column;
	gap: 0.45rem;
}

.field span {
	font-weight: 500;
	font-size: 0.92rem;
}

input,
select {
	border-radius: 12px;
	border: 1px solid var(--border-subtle);
	background: var(--surface-base);
	color: inherit;
	padding: 0.65rem 0.85rem;
	font-size: 0.95rem;
}

input:focus,
select:focus {
	outline: none;
	border-color: var(--accent);
	box-shadow: 0 0 0 2px rgba(20, 131, 255, 0.2);
}

small {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.grid {
	display: grid;
	gap: 1rem;
	grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.toggle {
	display: flex;
	gap: 0.75rem;
	padding: 0.85rem 1rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: var(--surface-hover);
	align-items: center;
}

.toggle input {
	width: 20px;
	height: 20px;
	margin: 0;
}

.toggle strong {
	display: block;
	font-size: 0.95rem;
	margin-bottom: 0.15rem;
}

.toggle span {
	font-size: 0.8rem;
	color: var(--color-text-tertiary);
}

.actions {
	display: flex;
	justify-content: flex-end;
}

.primary {
	border: none;
	background: var(--accent);
	color: #fff;
	border-radius: 12px;
	padding: 0.75rem 1.8rem;
	font-size: 0.95rem;
	box-shadow: 0 14px 26px rgba(20, 131, 255, 0.24);
	cursor: pointer;
	transition: transform 0.18s ease;
}

.primary:hover {
	transform: translateY(-1px);
}

@media (max-width: 720px) {
	.grid {
		grid-template-columns: 1fr;
	}
}
</style>
