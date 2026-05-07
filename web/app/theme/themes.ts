// app/theme/themes.ts
import type { ThemeDefinition, VuetifyOptions } from 'vuetify'

export const DEFAULT_THEME_NAME: 'light' | 'dark' = 'dark'

const sharedVars = {
  'high-emphasis-opacity': 0.92,
  'medium-emphasis-opacity': 0.62,
  'disabled-opacity': 0.34,
  'border-opacity': 0.08,
  'hover-opacity': 0.05,
  'focus-opacity': 0.1,
  'selected-opacity': 0.08,
  'activated-opacity': 0.12,
  'pressed-opacity': 0.12,
  'app-bar-elevation': 0,
  'card-elevation': 0,
  'dialog-elevation': 0,
} as const

export const dark: ThemeDefinition = {
  dark: true,
  colors: {
    'background': '#08090C',
    'surface': '#0E1015',
    'surface-bright': '#15171E',
    'surface-light': '#191C25',
    'surface-variant': '#1E222D',
    'on-background': '#E7EAF1',
    'on-surface': '#E0E3EC',
    'on-surface-variant': '#A8AEBE',
    'primary': '#7C9BFF',
    'on-primary': '#04060B',
    'primary-soft': '#DCE1FF',
    'primary-alt': '#162E69',
    'secondary': '#7DEBC8',
    'on-secondary': '#04140F',
    'secondary-soft': '#D8F2E6',
    'secondary-alt': '#0F6250',
    'accent': '#C68CFF',
    'on-accent': '#11031F',
    'accent-soft': '#EFE0FB',
    'accent-alt': '#3A174C',
    'success': '#4ED28A',
    'warning': '#F4C25B',
    'error': '#FF6B7C',
    'on-error': '#210207',
    'info': '#7DDDFF',
  },
  variables: {
    ...sharedVars,
    'border-color': '#262B3A',
  },
}

export const light: ThemeDefinition = {
  dark: false,
  colors: {
    'background': '#F6F7FB',
    'surface': '#FFFFFF',
    'surface-bright': '#FFFFFF',
    'surface-light': '#F0F2F8',
    'surface-variant': '#E6E9F2',
    'on-background': '#0E1015',
    'on-surface': '#0E1015',
    'on-surface-variant': '#3D4356',
    'primary': '#3554E0',
    'on-primary': '#FFFFFF',
    'primary-soft': '#DCE1FF',
    'primary-alt': '#162E69',
    'secondary': '#0FA37F',
    'on-secondary': '#FFFFFF',
    'secondary-soft': '#D8F2E6',
    'secondary-alt': '#0F6250',
    'accent': '#7E31B2',
    'on-accent': '#FFFFFF',
    'accent-soft': '#EFE0FB',
    'accent-alt': '#3A174C',
    'success': '#1E9D5C',
    'warning': '#A6740C',
    'error': '#D03A4F',
    'on-error': '#FFFFFF',
    'info': '#1565C0',
  },
  variables: {
    ...sharedVars,
    'border-color': '#C5CAD8',
  },
}

export const themes = {
  dark,
  light,
} as const

export type ThemeName = keyof typeof themes

const SURFACE = {
  rounded: 'lg',
  elevation: 0,
} as const

const BUTTON = {
  rounded: 'lg',
  elevation: 0,
  variant: 'flat',
} as const

const FIELD = {
  rounded: 'lg',
  variant: 'outlined',
  density: 'comfortable',
  hideDetails: 'auto',
} as const

const CHIPLIKE = {
  rounded: 'pill',
  variant: 'tonal',
} as const

export const defaults = {
  global: {
    rounded: 'lg',
    elevation: 0,
  },

  VAlert: { ...SURFACE, density: 'comfortable' },
  VAppBar: { rounded: 0, elevation: 0, color: 'transparent', flat: true, density: 'comfortable' },
  VBanner: SURFACE,
  VCard: SURFACE,
  VDialog: { rounded: 'lg' },
  VList: { rounded: 'lg', bgColor: 'transparent', density: 'comfortable' },
  VListItem: { rounded: 'lg' },
  VMenu: { rounded: 'lg', elevation: 0 },
  VOverlay: { rounded: 'lg' },
  VSheet: SURFACE,
  VSnackbar: SURFACE,
  VToolbar: { color: 'transparent', flat: true, density: 'comfortable' },

  VBtn: BUTTON,
  VBtnGroup: { rounded: 'lg', density: 'comfortable', variant: 'outlined' },
  VBtnToggle: { rounded: 'lg', density: 'comfortable', variant: 'outlined', divided: true },
  VFab: BUTTON,
  VChip: CHIPLIKE,

  VColorPicker: { elevation: 0, rounded: 'md' },
  VField: FIELD,
  VInput: { density: 'comfortable' },
  VNumberInput: { ...FIELD, controlVariant: 'default' },
  VTextField: FIELD,
  VTextarea: FIELD,
  VAutocomplete: FIELD,
  VCombobox: FIELD,
  VFileInput: FIELD,
  VSelect: { ...FIELD, flat: true },

  VAvatar: { rounded: 'lg' },
  VImg: { rounded: 'lg' },
  VProgressLinear: { rounded: true, height: 3 },
  VTooltip: { openDelay: 200 },
  VDivider: { opacity: 0.6 },
  VLabel: { class: 'text-caption' },
} as const

export const vuetifyOptions = {
  theme: {
    defaultTheme: DEFAULT_THEME_NAME,
    themes,
    variations: {
      colors: [
        'primary',
        'secondary',
        'accent',
        'error',
        'info',
        'success',
        'warning',
        'primary-soft',
        'secondary-soft',
        'accent-soft',
      ],
      lighten: 2,
      darken: 2,
    },
  },
  defaults,
} satisfies VuetifyOptions
