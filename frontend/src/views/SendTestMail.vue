<template>
    <form action="">
        <div class="modal-card" style="width: auto">
            <header class="modal-card-head">
                <p class="modal-card-title">{{ $t('templates.sendtest') }}</p>
            </header>
            <section class="modal-card-body">
                <p>{{ data.name }}</p>
                <b-field label="Email">
                    <b-input
                        type="email"
                        :value="email"
                        placeholder="Your email"
                        required>
                    </b-input>
                </b-field>
            </section>
            <footer class="modal-card-foot">
                <b-button
                    :label="$t('globals.buttons.close')"
                    @click="$emit('close')" />
                <b-button
                    :label="$t('globals.buttons.sendtest')"
                    type="is-primary" />
            </footer>
        </div>
    </form>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'SendTestMail',

  props: {
    email: String,
    data: Object,
  },

  methods: {
    close() {
      this.$emit('close');
    },
  },
});
</script>
