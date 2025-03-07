import type { Field } from '@alifd/next';
import { Icon, Loading, Grid, Switch, Dialog } from '@alifd/next';
import React from 'react';

import Translation from '../../components/Translation';
import locale from '../../utils/locale';
import './index.less';
import { If } from '../../components/If';

const { Col, Row } = Grid;
type Props = {
  title?: string | React.ReactNode;
  description?: string | React.ReactNode;
  children?: React.ReactNode;
  // if required is false, this will be effective
  closed?: boolean;
  // If set is true, in any case, the group is closed initially.
  initClose?: boolean;
  loading?: boolean;
  hasToggleIcon?: boolean;
  required?: boolean;
  field?: Field;
  jsonKey?: string;
  propertyValue?: any;
  alwaysShow?: boolean;
  disableAddon?: boolean;
  onChange?: (values: any) => void;
};

type State = {
  closed: boolean | undefined;
  enable?: boolean;
  checked: boolean;
};

class Group extends React.Component<Props, State> {
  dom: any;
  constructor(props: Props) {
    super(props);
    this.state = {
      closed: props.closed,
      enable: props.required,
      checked: false,
    };
  }

  toggleShowClass = () => {
    const { closed } = this.state;
    this.setState({
      closed: !closed,
    });
  };

  componentDidMount() {
    this.initSwitchState();
  }

  initSwitchState = () => {
    const {
      jsonKey = '',
      propertyValue = {},
      alwaysShow = false,
      required,
      closed,
      initClose,
    } = this.props;
    const findKey = Object.keys(propertyValue).find((item) => item === jsonKey);
    if (findKey || alwaysShow) {
      this.setState({ enable: true, closed: false || initClose, checked: true });
    } else if (required) {
      this.setState({ enable: true, closed: false || initClose, checked: true });
    } else {
      this.setState({ enable: false, closed: closed || initClose, checked: false });
    }
  };

  removeJsonKeyValue() {
    const { jsonKey = '', onChange } = this.props;
    const field: Field | undefined = this.props.field;
    if (field && onChange) {
      field.remove(jsonKey);
      const values = field.getValues();
      onChange(values);
    }
  }

  render() {
    const {
      title,
      description,
      children,
      hasToggleIcon,
      loading,
      required,
      disableAddon = false,
    } = this.props;
    const { closed, enable, checked } = this.state;
    return (
      <Loading visible={loading || false} style={{ width: '100%' }}>
        <div className="group-container">
          <If condition={title}>
            <div className="group-title-container">
              <Row>
                <Col span={'21'}>
                  <span className={`group-title ${required && 'required'}`}>{title}</span>
                  <div className="group-title-desc">{description}</div>
                </Col>
                <Col span={'3'} className="flexcenter">
                  <If condition={!required}>
                    <Switch
                      size="small"
                      defaultChecked={required}
                      checked={checked}
                      disabled={disableAddon}
                      onChange={(event: boolean) => {
                        if (event === true) {
                          this.setState({ enable: event, closed: false, checked: true });
                        } else if (event === false) {
                          Dialog.confirm({
                            type: 'confirm',
                            content: (
                              <Translation>
                                The configuration will be reset if the switch is turned off. Are you
                                sure want to do this?
                              </Translation>
                            ),
                            onOk: () => {
                              this.setState({ enable: event, closed: false, checked: false });
                              this.removeJsonKeyValue();
                            },
                            locale: locale().Dialog,
                          });
                        }
                      }}
                    />
                  </If>
                  <If condition={enable && hasToggleIcon}>
                    <Icon
                      onClick={this.toggleShowClass}
                      className="icon"
                      type={closed ? 'arrow-down' : 'arrow-up'}
                    />
                  </If>
                </Col>
              </Row>
            </div>
          </If>
          <If condition={enable}>
            <div className={`group-box ${closed ? 'disable' : ''}`}>{children}</div>
          </If>
        </div>
      </Loading>
    );
  }
}

export default Group;
