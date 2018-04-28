require(['gitbook', 'jquery'], function(gitbook, $) {
    var buttonsId = [],
        $codes,
        themeApi;

    // Default themes
    var THEMES = [
        {
            config: 'light',
            text: 'Light',
            id: 0
        },
        {
            config: 'dark',
            text: 'Dark',
            id: 3
        }
    ];

    // Instantiate localStorage
    function init(config) {
        themeApi = gitbook.storage.get('themeApi', {
            split:       config.split,
            currentLang: null
        });
    }

    // Update localStorage settings
    function saveSettings() {
        gitbook.storage.set('themeApi', themeApi);
        updateDisplay();
    }

    // Update display
    function updateDisplay() {
        // Update layout
        $('.book').toggleClass('two-columns', themeApi.split);

        // Update code samples elements
        $codes = $('.api-method-sample');
        // Display corresponding code snippets
        $codes.each(function() {
            // Show corresponding
            var hidden = !($(this).data('lang') == themeApi.currentLang);
            $(this).toggleClass('hidden', hidden);
        });
    }

    // Update code tabs
    function updateCodeTabs() {
        // Remove languages buttons
        gitbook.toolbar.removeButtons(buttonsId);
        buttonsId = [];

        // Update code snippets elements
        $codes = $('.api-method-sample');

        // Recreate languages buttons
        var languages = [],
            hasCurrentLang = false;

        $codes.each(function() {
            var isDefault = false,
                codeLang  = $(this).data('lang'),
                codeName  = $(this).data('name'),
                exists,
                found;

            // Check if is current language
            if (codeLang == themeApi.currentLang) {
                hasCurrentLang = true;
                isDefault = true;
            }

            // Check if already added
            exists = $.grep(languages, function(language) {
                return language.name == codeName;
            });

            found = !!exists.length;

            if (!found) {
                // Add language
                languages.push({
                    name: codeName,
                    lang: codeLang,
                    default: isDefault
                });
            }
        });

        // Set languages in good order
        languages.reverse();
        $.each(languages, function(i, language) {
            // Set first (last in array) language as active if no default
            var isDefault = language.default || (!hasCurrentLang && i == (languages.length - 1)),
                buttonId;

            // Create button
            buttonId = gitbook.toolbar.createButton({
                text: language.name,
                position: 'right',
                className: 'lang-switcher' + (isDefault? ' active': ''),
                onClick: function(e) {
                    // Update language
                    themeApi.currentLang = language.lang;
                    saveSettings();

                    // Update active button
                    $('.btn.lang-switcher.active').removeClass('active');
                    $(e.currentTarget).addClass('active');
                }
            });

            // Add to list of buttons
            buttonsId.push(buttonId);

            // Set as current language if is default
            if (isDefault) {
                themeApi.currentLang = language.lang;
            }
        });
    }

    // Initialization
    gitbook.events.bind('start', function(e, config) {
        var opts = config['theme-api'];

        // Create layout button in toolbar
        gitbook.toolbar.createButton({
            icon: 'fa fa-columns',
            label: 'Change Layout',
            onClick: function() {
                // Update layout
                themeApi.split = !themeApi.split;
                saveSettings();
            }
        });

        // Initialize themes
        gitbook.fontsettings.setThemes(THEMES);

        // Set to configured theme
        gitbook.fontsettings.setTheme(opts.theme);

        // Init current settings
        init(opts);
    });

    // Update state
    gitbook.events.on('page.change', function() {
        updateCodeTabs();
        // updateComments();
        updateDisplay();
    });

    // Comments toggled event
    gitbook.events.on('comment.toggled', function(e, $from, open) {
        // If triggering element is in a definition
        if (!!$from.parents('.api-method-definition').length) {
            // Add class to wrapper only if comments are open and in two-columns mode
            var $wrapper = gitbook.state.$book.find('.page-wrapper');
            $wrapper.toggleClass('comments-open-from-definition', open && themeApi.split);
        }
    });
});
