import { Grid, Icon, Select, Input, Button } from '@alifd/next';
import React from 'react';

import './index.less';
import type { ShowMode } from '../..';
import type { Env } from '../../../../interface/env';
import type { Project } from '../../../../interface/project';
import locale from '../../../../utils/locale';
import i18n from '../../../../i18n';
import { AiOutlineSearch } from 'react-icons/ai';

const { Row, Col } = Grid;

type Props = {
  dispatch: ({}) => {};
  projects?: Project[];
  envs?: Env[];
  getApplications: (params: any) => void;
  setMode: (mode: ShowMode) => void;
  showMode: ShowMode;
};

type State = {
  projectValue: string;
  targetValue: string;
  inputValue: string;
  envValue: string;
};

class SelectSearch extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      projectValue: '',
      targetValue: '',
      envValue: '',
      inputValue: '',
    };
    this.onChangeProject = this.onChangeProject.bind(this);
    this.onChangeTarget = this.onChangeTarget.bind(this);
    this.handleChangName = this.handleChangName.bind(this);
  }

  onChangeProject(e: string) {
    this.setState(
      {
        projectValue: e,
      },
      () => {
        this.getApplications();
      }
    );
  }

  onChangeTarget(e: string) {
    this.setState(
      {
        targetValue: e,
      },
      () => {
        this.getApplications();
      }
    );
  }

  handleChangName(e: string) {
    this.setState({
      inputValue: e,
    });
  }

  onChangeEnv = (e: string) => {
    this.setState(
      {
        envValue: e,
      },
      () => {
        this.getApplications();
      }
    );
  };

  handleClickSearch = () => {
    this.getApplications();
  };

  getApplications = async () => {
    const { projectValue, inputValue, envValue } = this.state;
    const params = {
      project: projectValue,
      query: inputValue,
      env: envValue,
    };
    this.props.getApplications(params);
  };

  render() {
    const { projects, envs, showMode } = this.props;
    const { projectValue, inputValue, envValue } = this.state;

    const projectPlaceholder = i18n.t('Search by Project').toString();
    const appPlaceholder = i18n.t('Search by name and description etc').toString();
    const envPlaceholder = i18n.t('Search by Environment').toString();
    const projectSource = projects?.map((item) => {
      return {
        label: item.alias || item.name,
        value: item.name,
      };
    });

    const envSource = envs?.map((env) => {
      return {
        label: env.alias || env.name,
        value: env.name,
      };
    });
    return (
      <Row className="app-select-wrapper border-radius-8" wrap={true}>
        <Col xl={6} m={8} s={12} xxs={24} style={{ padding: '0 8px' }}>
          <Select
            locale={locale().Select}
            mode="single"
            size="large"
            onChange={this.onChangeProject}
            dataSource={projectSource}
            placeholder={projectPlaceholder}
            className="item"
            hasClear
            value={projectValue}
          />
        </Col>
        <Col xl={6} m={8} s={12} xxs={24} style={{ padding: '0 8px' }}>
          <Select
            locale={locale().Select}
            mode="single"
            size="large"
            onChange={this.onChangeEnv}
            dataSource={envSource}
            placeholder={envPlaceholder}
            className="item"
            hasClear
            value={envValue}
          />
        </Col>
        <Col xl={6} m={8} s={12} xxs={24} style={{ padding: '0 8px' }}>
          <Input
            innerAfter={<AiOutlineSearch onClick={this.handleClickSearch} style={{ margin: 4 }} />}
            hasClear
            size="large"
            placeholder={appPlaceholder}
            onChange={this.handleChangName}
            onPressEnter={this.handleClickSearch}
            value={inputValue}
            className="item"
          />
        </Col>
        <Col xl={6} className="flexboth">
          <div className="padding16">
            <Button type={'secondary'} onClick={() => this.getApplications()}>
              <Icon type="refresh" />
            </Button>
          </div>
          <div className="show-mode padding16">
            <Button.Group>
              <Button type={showMode == 'card' ? 'primary' : 'secondary'} onClick={() => this.props.setMode('card')}>
                Card
              </Button>
              <Button type={showMode == 'table' ? 'primary' : 'secondary'} onClick={() => this.props.setMode('table')}>
                Table
              </Button>
            </Button.Group>
          </div>
        </Col>
      </Row>
    );
  }
}

export default SelectSearch;
