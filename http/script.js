define(['jquery'], function($) {
    var CustomWidget = function () {
        var self = this, // для доступа к объекту из методов
            system = self.system(), //Данный метод возвращает объект с переменными системы.
            langs = self.langs;  //Объект локализации с данными из файла локализации (папки i18n)

        var params = [
            {name:'name1',
                id: 'id1'},
            {name:'name2',
                id: 'id2'},
            {name:'name3',
                id: 'id3'}
        ]; //массив данных, передаваемых для шаблона

        var template = '<div><ul>'+
            '{% for person in names %}'+
            '<li>Name : {{ person.name }}, id: {{ person.id }}</li>'+
            '{% endfor %}'+
            '</ul></div>';

        console.log(self.render({data : template},// передаем шаблон
            {names: params}));

        return this;
    };
    return CustomWidget;
});