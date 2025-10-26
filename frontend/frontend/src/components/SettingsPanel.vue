<script lang="ts" setup>
import {computed, reactive, ref, watch} from 'vue';
import PanelShell from './base/PanelShell.vue';
import AppButton from './base/AppButton.vue';
import SettingsNav from './settings/SettingsNav.vue';
import SettingsSection from './settings/SettingsSection.vue';
import SettingsApiSection from './settings/SettingsApiSection.vue';
import SettingsModelSection from './settings/SettingsModelSection.vue';
import SettingsBehaviorSection from './settings/SettingsBehaviorSection.vue';
import SettingsPromptSection from './settings/SettingsPromptSection.vue';
import SettingsHotkeySection from './settings/SettingsHotkeySection.vue';
import SettingsThemeSection from './settings/SettingsThemeSection.vue';
import {provideSettingsForm} from './settings/useSettingsForm';
import {useSettingsNavigation} from '../composables/useSettingsNavigation';
import type {SettingsState} from '../types';
import {defaultSettingsState} from '../types';

const props = defineProps<{
	settings: SettingsState;
	apiKeyMissing: boolean;
}>();

const emit = defineEmits<{
	(event: 'submit', value: SettingsState): void;
}>();

const form = reactive<SettingsState>({...props.settings});
provideSettingsForm(form);

const navigation = useSettingsNavigation();
const categories = navigation.settingsCategories;
const activeCategoryValue = computed(() => navigation.activeCategory.value);
const currentCategoryValue = computed(() => navigation.currentCategory.value);
const isSectionVisible = navigation.isSectionVisible;
const isSectionExpanded = navigation.isSectionExpanded;
const toggleSection = navigation.toggleSection;
const activateCategory = navigation.activateCategory;

const validationError = ref<string | null>(null);

const showTranslateApiFields = computed(() => !form.useVisionForTranslation);
const hasVisionKey = computed(() => Boolean(form.visionApiKeyOverride?.trim() || form.apiKeyOverride?.trim()));

watch(
	() => props.settings,
	(next) => {
		Object.assign(form, next);
	},
	{deep: true},
);

watch(
	[
		() => form.useVisionForTranslation,
		() => form.translateModel,
		() => form.visionModel,
		() => form.visionApiKeyOverride,
		() => form.apiKeyOverride,
	],
	() => {
		validationError.value = null;
	},
);

function handleSubmit() {
	validationError.value = null;
	const translateModel = form.translateModel?.trim();
	const visionModel = form.visionModel?.trim();
	if (form.useVisionForTranslation) {
		if (!visionModel) {
			validationError.value = '视觉直出模式下需设置视觉模型名称，或关闭该模式。';
			return;
		}
		if (!hasVisionKey.value) {
			validationError.value = '请配置视觉专用或通用 API Key，以便正常调用视觉模型。';
			return;
		}
	} else if (!translateModel) {
		validationError.value = '请填写翻译模型名称，或启用视觉直出模式。';
		return;
	}
	emit('submit', {...form});
}

function resetForm() {
	Object.assign(form, defaultSettingsState());
	validationError.value = null;
}
</script>

<template>
	<PanelShell :max-width="1100">
		<form class="settings" @submit.prevent="handleSubmit">
			<div class="settings-layout">
					<SettingsNav
						:categories="categories"
						:active-category="activeCategoryValue"
						:validation-error="validationError"
						@select="activateCategory"
					>
					<template #actions>
						<div class="settings-nav-actions">
							<AppButton type="submit">保存设置</AppButton>
							<AppButton variant="ghost" type="button" class="settings-actions" @click="resetForm">恢复默认</AppButton>
						</div>
					</template>
				</SettingsNav>
				<div class="settings-content">
						<header class="settings-content__intro">
							<h2>{{ currentCategoryValue.label }}</h2>
							<p>{{ currentCategoryValue.description }}</p>
						</header>

					<SettingsSection
						v-if="isSectionVisible('api')"
						title="接口与凭证"
						:description="props.apiKeyMissing ? '未检测到 API Key，请录入可用凭证以启用服务。' : '凭证已配置，可直接使用截图与翻译能力。'"
						:expanded="isSectionExpanded('api')"
						@toggle="toggleSection('api')"
					>
						<SettingsApiSection
							:show-translate-fields="showTranslateApiFields"
							:api-key-missing="props.apiKeyMissing"
						/>
					</SettingsSection>

					<SettingsSection
						v-if="isSectionVisible('models')"
						title="模型能力"
						:expanded="isSectionExpanded('models')"
						@toggle="toggleSection('models')"
					>
						<SettingsModelSection />
					</SettingsSection>

					<SettingsSection
						v-if="isSectionVisible('behavior')"
						title="工作流行为"
						description="控制翻译完成后的动作逻辑，保持团队协作节奏。"
						:expanded="isSectionExpanded('behavior')"
						@toggle="toggleSection('behavior')"
					>
						<SettingsBehaviorSection />
					</SettingsSection>

					<SettingsSection
						v-if="isSectionVisible('prompts')"
						title="提示词管理"
						description="针对视觉直出与文本兜底流程，定制提示词让上下文更贴合业务术语。"
						:expanded="isSectionExpanded('prompts')"
						@toggle="toggleSection('prompts')"
					>
						<SettingsPromptSection />
					</SettingsSection>

					<SettingsSection
						v-if="isSectionVisible('hotkey')"
						title="热键偏好"
						description="统一热键与交互方式，保持操作一致性。"
						:expanded="isSectionExpanded('hotkey')"
						@toggle="toggleSection('hotkey')"
					>
						<SettingsHotkeySection />
					</SettingsSection>

					<SettingsSection
						v-if="isSectionVisible('theme')"
						title="界面主题"
						description="设置主题与视觉偏好，营造舒适的使用体验。"
						:expanded="isSectionExpanded('theme')"
						@toggle="toggleSection('theme')"
					>
						<SettingsThemeSection />
					</SettingsSection>
				</div>
			</div>
		</form>
	</PanelShell>
</template>

<style scoped>
.settings {
	width: 100%;
}

.settings-layout {
	display: grid;
	grid-template-columns: 260px minmax(0, 1fr);
	gap: 1.75rem;
	align-items: flex-start;
	padding-top: 1.25rem;
}

.settings-content {
	display: flex;
	flex-direction: column;
	gap: 1.35rem;
}

.settings-content__intro {
	display: flex;
	flex-direction: column;
	gap: 0.3rem;
	margin-bottom: 0.3rem;
}

.settings-content__intro h2 {
	margin: 0;
	font-size: 1.32rem;
	font-weight: 600;
}

.settings-content__intro p {
	margin: 0;
	color: var(--color-text-tertiary);
	font-size: 0.9rem;
}

.settings-nav-actions {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
}

.settings-actions {
	display: flex;
	justify-content: flex-end;
}

@media (max-width: 960px) {
	.settings-layout {
		grid-template-columns: 1fr;
	}

	.settings-content__intro h2 {
		font-size: 1.2rem;
	}
}
</style>