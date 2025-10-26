import {computed, reactive, ref, watch} from 'vue';
import {
	SECTION_STORAGE_KEY,
	sectionDefaults,
	settingsCategories,
	type CategoryKey,
	type SectionKey,
	type SettingsCategory,
} from '../config/settings';

interface SectionState {
	[key: string]: boolean;
}

function loadSectionState(): SectionState | null {
	if (typeof window === 'undefined') {
		return null;
	}
	try {
		const raw = window.localStorage.getItem(SECTION_STORAGE_KEY);
		return raw ? JSON.parse(raw) : null;
	} catch (error) {
		console.warn('恢复折叠状态失败', error);
		return null;
	}
}

function persistSectionState(value: SectionState) {
	if (typeof window === 'undefined') {
		return;
	}
	try {
		window.localStorage.setItem(SECTION_STORAGE_KEY, JSON.stringify(value));
	} catch (error) {
		console.warn('保存折叠状态失败', error);
	}
}

function ensureOneExpanded(targetSections: SectionKey[], sections: Record<SectionKey, boolean>) {
	if (!targetSections.length) {
		return;
	}
	const hasExpanded = targetSections.some((key) => sections[key]);
	if (!hasExpanded) {
		sections[targetSections[0]] = true;
	}
}

export function useSettingsNavigation() {
	const sections = reactive<Record<SectionKey, boolean>>({...sectionDefaults});
	const activeCategory = ref<CategoryKey>('integration');

	const storedSections = loadSectionState();
	if (storedSections) {
		(Object.keys(sectionDefaults) as SectionKey[]).forEach((key) => {
			const value = storedSections[key];
			if (typeof value === 'boolean') {
				sections[key] = value;
			}
		});
	}

	const currentCategory = computed<SettingsCategory>(() => {
		const found = settingsCategories.find((item) => item.key === activeCategory.value);
		return found ?? settingsCategories[0];
	});

	const visibleSections = computed<SectionKey[]>(() => currentCategory.value.sections);

	function activateCategory(name: CategoryKey) {
		if (activeCategory.value === name) {
			return;
		}
		activeCategory.value = name;
		const target = settingsCategories.find((item) => item.key === name);
		if (target) {
			ensureOneExpanded(target.sections, sections);
		}
	}

	function isCategoryActive(name: CategoryKey) {
		return activeCategory.value === name;
	}

	function isSectionVisible(name: SectionKey) {
		return visibleSections.value.includes(name);
	}

	function isSectionExpanded(name: SectionKey) {
		return sections[name] ?? true;
	}

	function toggleSection(name: SectionKey) {
		sections[name] = !isSectionExpanded(name);
	}

	watch(
		sections,
		(next) => {
			persistSectionState({...next});
		},
		{deep: true},
	);

	return {
		sections,
		settingsCategories,
		currentCategory,
		visibleSections,
		activeCategory,
		activateCategory,
		isCategoryActive,
		isSectionVisible,
		isSectionExpanded,
		toggleSection,
	};
}

export type SettingsNavigation = ReturnType<typeof useSettingsNavigation>;
