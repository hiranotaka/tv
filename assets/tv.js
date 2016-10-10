function updatePositions() {
    $('td.main-hour').css('left', $(window).scrollLeft() + 'px');
    $('td.main-program').css('top', $(window).scrollTop() + 'px');
}

function updateSelectedEvent() {
    var params = { 'want-event': true };
    $('.event').parent().load(window.location.href + ' .event',
			      $.param(params));
}

$(window).scroll(function () {
    updatePositions();
});
updatePositions();

$(window).submit(function(event) {
    var target = $(event.target);
    var method = target.prop('method');
    var action = target.prop('action');
    var params = method == 'get' ?  null : target.serializeArray();
    var url = method == 'get' ? './?' + target.serialize() : action;
    window.history.pushState(null, null, url);
    $('.main').parent().load(
	url + ' .main', params, function() {
	    updatePositions();
	});
    $('.event :input').prop('disabled', true);
    event.preventDefault();
});

$(window).click(function(event) {
    var target = $(event.target);
    var href = target.prop('href');
    if (href) {
	window.history.pushState(null, null, href);
	updateSelectedEvent();
	event.preventDefault();
    }
});

$(window).bind('popstate', function() {
    updateSelectedEvent();
});
