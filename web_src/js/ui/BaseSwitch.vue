<template>
  <label
    class="wrapper"
    role="switch"
    :for="id">
    <div class="switch">
      <input
        type="checkbox"
        :id="id"
        :disabled="disabled"
        :checked="modelValue"
        @change="handleChange"
      />
      <span class="slider round"></span>
    </div>
    <span v-if="label" class="label">{{ label }}</span>
  </label>
</template>

<script>
import { ref, onMounted, computed } from 'vue';

export default {
  props: {
    id: {
      type: String,
      default: '',
    },
    name: {
      type: String
    },
    label: {
      type: String
    },
    disabled: {
      type: Boolean,
      default: false
    },
    modelValue: {
      type: Boolean,
      required: true,
    },
  },
  emits: ['update:modelValue'],
  methods: {
    handleChange(event) {
      this.$emit('update:modelValue', event.target.checked); // Генерируем событие
    },
  },
  setup(props) {
    const generatedId = ref('');

    // Генерация уникального ID, если он не передан
    onMounted(() => {
      if (!props.id) {
        generatedId.value = `switch-${Math.random().toString(36).substring(2, 9)}`;
      }
    });

    const id = computed(() => props.id || generatedId.value);

    return {
      id,
      modelValue: props.modelValue, // Используем переданный modelValue или сгенерированный
    };
  },
};
</script>

<style scoped>
.wrapper{
  display: flex;
  align-items: center;
  column-gap: 8px;
  cursor: pointer;
}

/* The switch - the box around the slider */
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 20px;
}

/* Hide default HTML checkbox */
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

/* The slider */
.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgb(211, 219, 223);
  -webkit-transition: .4s;
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  -webkit-transition: .4s;
  transition: .4s;
}

input:checked + .slider {
  background-color: rgb(25, 118, 210);
}

input:focus + .slider {
  box-shadow: 0 0 1px rgb(25, 118, 210);
}

input:checked + .slider:before {
  -webkit-transform: translateX(20px);
  -ms-transform: translateX(20px);
  transform: translateX(20px);
}

/* Rounded sliders */
.slider.round {
  border-radius: 20px;
}

.slider.round:before {
  border-radius: 50%;
}
</style>
