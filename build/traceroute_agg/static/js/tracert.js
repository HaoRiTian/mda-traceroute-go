
window.onload = function() {
    $.ajax({
        type: "get",
        url: "/api/nodes",
        data: {},
        dataType: "json",
        success: function (data) {
            var region = document.getElementById("groups");
            addRadioButton({
                element: region,
                name: "group",
                id: "group",
                value: "All",
                checked: true,
                onclick: function () {

                }
            });

            if (data != null && data.data != null) {
                for (var j = 0; j < data.data.length; j++) {
                    console.log(data.data[j])
                    addRadioButton({
                        element: region,
                        name: "group",
                        id: "group",
                        value: data.data[j],
                        checked: false,
                        onclick: function () {

                        }
                    });
                };
            }
        }
    })
}

function listMap(map, keys) {
    var arr = new Array()
    for (let i = 0; i < keys.length; i++) {
        let recordArr = map[keys[i]]
        for (let j=0; j<recordArr.length; j++) {
            arr.push(recordArr[j])
        }
    }
    return arr
}


function sortMapKey(obj, desc = false) {
    let keys = new Array()
    for (let key in obj) {
        keys.push(key)
    }
    if (!desc) {
        keys.sort(function (a,b) {
            return a-b
        })
    } else {
        keys.sort(function (a,b) {
            return b-a
        })
    }
    return keys
}


/* 添加单选框
    var region = document.getElementById("add");
    addRadioButton({
        element : region,
        name : "new",
        id : "new",
        value : 0,
        checked : false,
        onclick : function(){
            alert(this.value);
        }
    });
 */
function addRadioButton(radio) {
    // if (radio.element == null ||
    //     radio.name == null ||
    //     radio.value == null ||
    //     (radio.onclick != null && typeof radio.onclick != "function")) {
    //     throw new Error("ErrorArgument");
    // }
    const options = {
        element: radio.element,
        name: radio.name,
        id: radio.id == null ? "" : radio.id,
        value: radio.value,
        checked: !(radio.checked == null || !radio.checked),
        onclick: radio.onclick == null ? function () {
        } : radio.onclick,
        css: radio.css == null ? "" : radio.css
    };

    //声明内部函数
    function addEvent(obj, type, fn) {
        if (obj.addEventListener) obj.addEventListener(type, fn, false);
        else if (obj.attachEvent) {
            obj["e" + type + fn] = fn;
            obj[type + fn] = function () {
                obj["e" + type + fn](window.event);
            }
            obj.attachEvent("on" + type, obj[type + fn]);
        }
    }

    //动态生成单选框
    // var strInput = "input type='radio' name='" + options.name + "'";

    var op = document.createElement('input');
    op.type = "radio";
    op.name = options.name;
    op.id = options.id;
    op.value = options.value;
    op.className = options.css;
    op.innerText = options.value;

    addEvent(op, "click", function () {
        options.onclick.call(this);
    });

    //添加到DOM树
    radio.element.appendChild(op);

    var txt = document.createTextNode(options.value);
    radio.element.appendChild(txt);

    //设置选中
    op.checked = options.checked;
}