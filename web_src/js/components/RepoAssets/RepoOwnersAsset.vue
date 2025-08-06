<template>
  <div>
    <h4 class="ui top attached header">
      {{ t('owners_settings') }}
    </h4>
    <div class="ui attached segment owners">
      <form class="ui form" @submit.prevent="handleSubmit">
        <input type="hidden" name="csrf_token" :value="csrfToken" />
        <input type="hidden" name="action" value="owners">
        <div class="field">
          <div class="ui checkbox">
            <input
              id="approval_checkbox"
              type="checkbox"
              v-model="approvalStatus"
            />
            <label>{{ t('owners_approval_status') }}</label>
          </div>
        </div>
        <div class="owners">
          <label for="amount_users">{{ t('owners_approve_amount') }}</label>
          <input
            class="owners-input"
            id="amount_users"
            v-model="amountUsers"
            @input="validateInput"
            required
          />
          <label>{{ t('owners_setting_hint') }}</label>
          <div v-if="errorMessage" class="error-message" style="color: red;">
            {{ errorMessage }}
          </div>
        </div>
        <div class="field">
          <button
            id="save_button"
            class="sc-button sc-button_primary"
            :disabled="!isFormValid"
          >
            {{ t('save_application') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';


const approvalStatus = ref(false);
const amountUsers = ref('');
const errorMessage = ref('');
const repoLink = ref('');


const { t } = useI18n({
  inheritLocale: true,
  useScope: 'local'
});


onMounted(() => {
  const appElement = document.getElementById('repo-owners-asset');
  if (appElement && appElement.dataset) {
    repoLink.value = appElement.dataset.repoLink || '';
  }
});

const validateInput = () => {
  amountUsers.value = amountUsers.value.replace(/[^1-9*]/g, '');
  const regex = /^(\*|[1-9]{1,2}|100)$/;
  if (!regex.test(amountUsers.value)) {
    errorMessage.value = t('repo-error-message', 'Введите число от 1 до 100 или символ *');
  } else {
    errorMessage.value = '';
  }
};

const isFormValid = computed(() => {
  return approvalStatus.value && amountUsers.value.trim() !== '' && !errorMessage.value;
});

const handleSubmit = async () => {
  const { csrfToken } = window.config;
  if (isFormValid.value) {
    try {
      const formData = new FormData();
      formData.append('csrf_token', csrfToken);
      formData.append('action', 'owners');
      formData.append('approval_status', approvalStatus.value);
      formData.append('amount_users', amountUsers.value);

      const baseUrl = repoLink.value.replace(/['"]/g, ''); // Удаляем кавычки
      const url = `${baseUrl}/settings`;

      // Используем repoLink из data-атрибута
      const response = await fetch(`${url}`, {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        console.log('Форма успешно отправлена');
      } else {
        console.error('Ошибка при отправке формы:', response.statusText);
      }
    } catch (error) {
      console.error('Ошибка сети:', error);
    }
  } else {
    console.error('Форма не прошла валидацию');
  }
};
</script>

<style scoped>
label {
  color: var(--sc-color-text-secondary);
}
.error-message {
  font-size: 0.9em;
  margin-top: 5px;
}
input.error {
  border: 2px solid red;
}
</style>

<i18n>
{
  "en-US": {
    "owners_approval_status": "Approval of the code owners is required",
    "owners_approve_amount": "Required approval number",
    "owners_settings": "Code Owners Approval",
    "owners_setting_hint": "Use * to select all the code owners",
    "save_application": "Save",
    "repo-error-message": "Enter number from 1 to 100 or * symbol"
  },
  "ru-RU": {
    "owners_approval_status": "Необходимо одобрение владельцев кода",
    "owners_approve_amount": "Необходимое число одобрений",
    "owners_settings": "Одобрение владельцев кода",
    "owners_setting_hint": "Чтобы выбрать всех владельцев кода, используйте *",
    "save_application": "Сохранить",
    "repo-error-message": "Введите число от 1 до 100 или символ *"
  }
}
</i18n>
